(() => {
  const authKey = 'drukleague-authenticated';
  const roleKey = 'drukleague-role';
  const usernameKey = 'drukleague-username';
  const safeRole = sanitizeRole;
  const safeUsername = sanitizeUsername;
  const safeUUID = sanitizeUUID;
  const token = () => sessionStorage.getItem(authKey) === 'true';
  const defaultApiRoot = location.protocol === 'file:' ? 'http://127.0.0.1:8080/api' : `${location.origin}/api`;
  const configuredApiRoot = (() => {
    try {
      const candidate = new URL(window.DRUKLEAGUE_API_BASE || defaultApiRoot, location.href);
      const sameOrigin = candidate.origin === location.origin && location.origin !== 'null';
      const localDevelopmentOrigin = ['http://127.0.0.1:8080','http://localhost:8080'].includes(candidate.origin);
      if ((!sameOrigin && !localDevelopmentOrigin) || candidate.pathname.replace(/\/+$/, '') !== '/api' || candidate.search || candidate.hash) return new URL(defaultApiRoot);
      return candidate;
    } catch (_) { return new URL(defaultApiRoot); }
  })();
  const apiRoot = configuredApiRoot.href.replace(/\/+$/, '');
  const headers = () => ({'Content-Type':'application/json'});
  async function request(path, options={}) {
    const safeURL = safeAPIURL(apiRoot, path, options.method);
    if (!safeURL) throw new Error('Invalid API request path.');
    const response = await fetch(safeURL, {...options, credentials:'same-origin', headers:{...headers(), ...(options.headers||{})}});
    const data = await response.json().catch(()=>({}));
    if (!response.ok) throw new Error(data.error || `Request failed (${response.status})`);
    return data;
  }
  const initials = (name) => name.split(/\s+/).map(part=>part[0]||'').join('').slice(0,2).toUpperCase();
  function syncRoleIdentity() {
    const signedIn = Boolean(token());
    const identity = roleIdentity(
      signedIn ? (sessionStorage.getItem(usernameKey) || '') : '',
      signedIn ? currentRole() : 'viewer',
      signedIn
    );
    document.getElementById('profile-avatar').textContent = identity.badge;
    document.getElementById('profile-name').textContent = identity.displayName;
    document.getElementById('profile-role').textContent = identity.roleLabel;
    const authButton = document.getElementById('auth-button');
    authButton.textContent = identity.badge;
    authButton.setAttribute('aria-label', signedIn ? `Sign out ${identity.displayName}` : 'Sign in to DrukLeague');
    document.getElementById('profile-button').setAttribute('aria-label', signedIn ? `${identity.displayName}, ${identity.roleLabel}. Sign out.` : 'Public viewer, Viewer role. Sign in.');
  }
  async function refresh() {
    if (!token()) return;
    try {
      const [remoteTeams, remoteTournaments, remoteMatches] = await Promise.all([request('/teams'),request('/tournaments'),request('/matches')]);
      const standingsTournamentID=safeUUID(remoteTournaments[0]?.id);
      const table=standingsTournamentID?await request(`/standings?tournament_id=${encodeURIComponent(standingsTournamentID)}`):[];
      const byTeam=new Map(table.map(row=>[row.team_id,row]));
      teams.splice(0,teams.length,...remoteTeams.map(t=>{const row=byTeam.get(t.id)||{};return {id:t.id,name:t.name,short:t.short_code||initials(t.name),color:'',dz:t.dzongkhag,played:row.played||0,w:row.won||0,d:row.drawn||0,l:row.lost||0,gf:row.goals_for||0,ga:row.goals_against||0,gd:row.goal_difference===undefined?'0':String(row.goal_difference),pts:row.points||0,form:[]}}));
      tournaments.splice(0,tournaments.length,...remoteTournaments.map(t=>({id:t.id,name:t.name,sport:t.sport,season:t.season,status:t.status==='completed'?'Completed':t.status==='in_progress'?'In progress':'Registration open',teams:0,fixtures:0,fill:0,theme:'',description:`${t.sport} · ${t.season}`,venue:t.venue,dz:t.dzongkhag})));
      fixtureData.splice(0,fixtureData.length,...remoteMatches.map(m=>({id:m.id,tournamentId:m.tournament_id,time:m.kickoff?new Date(m.kickoff).toLocaleTimeString([],{hour:'2-digit',minute:'2-digit'}):'TBD',period:m.status.toUpperCase(),home:m.home,h:initials(m.home),away:m.away,a:initials(m.away),aColor:'warm',venue:m.venue,status:m.status==='completed'?'COMPLETED':m.status==='live'?'LIVE':'UPCOMING',score:m.result_home!==null?`${m.result_home} — ${m.result_away}`:m.live_home!==null?`${m.live_home} — ${m.live_away}`:'',official:'Assigned official',matchResult:m.status==='completed'?{homeGoals:m.result_home,awayGoals:m.result_away}:null})));
      render(activePage);
    } catch (error) { toast(`Could not load saved project data: ${error.message}`); }
  }
  async function showLogin() {
    if (token()) {
      try { await request('/auth/logout',{method:'POST'}); } catch (error) { toast(error.message); return; }
      sessionStorage.removeItem(authKey); sessionStorage.removeItem(roleKey); sessionStorage.removeItem(usernameKey); syncRoleIdentity(); render('dashboard');
      toast('Signed out.'); return;
    }
    document.getElementById('modal-root').innerHTML=`<div class="modal-backdrop" data-action="dismiss-modal"><section class="modal" role="dialog" aria-modal="true" aria-labelledby="login-title"><header class="modal-head"><div><h2 id="login-title">Sign in to DrukLeague</h2><p>Connect to your saved tournaments and match data.</p></div><button class="modal-close" type="button" data-action="dismiss-modal" aria-label="Close">×</button></header><form id="login-form"><div class="modal-body"><div class="form-grid"><div class="form-field full"><label for="login-user">Username</label><input id="login-user" name="username" autocomplete="username" required></div><div class="form-field full"><label for="login-password">Password</label><input id="login-password" name="password" type="password" autocomplete="current-password" required></div></div></div><footer class="modal-footer"><button type="button" class="button" data-action="dismiss-modal">Cancel</button><button type="submit" class="button button-primary">Sign in</button></footer></form></section></div>`;
    document.getElementById('login-user').focus();
  }
  document.getElementById('auth-button').addEventListener('click',showLogin);
  document.getElementById('profile-button').addEventListener('click',showLogin);
  document.addEventListener('submit', async (event) => {
    const form=event.target;
    if (form.id==='login-form') {
      event.preventDefault();
      try { const result=await request('/auth/login',{method:'POST',body:JSON.stringify({username:form.elements.username.value,password:form.elements.password.value})});const user=result.user||{};const username=safeUsername(user.username);sessionStorage.setItem(authKey,'true');sessionStorage.setItem(roleKey,safeRole(user.role));sessionStorage.setItem(usernameKey,username);syncRoleIdentity();document.getElementById('modal-root').innerHTML='';render(activePage);toast(`Signed in as ${username||'user'}.`);await refresh(); }
      catch(error){
        toast(error.message);
        const modal=document.querySelector('#login-form .modal-body');
        if (modal) {
          let message=document.getElementById('login-error');
          if (!message) {
            message=document.createElement('p');
            message.id='login-error';
            message.setAttribute('role','alert');
            message.style.color='#a34f3f';
            modal.prepend(message);
          }
          message.textContent=`Could not sign in: ${error.message}`;
        }
      }
      return;
    }
    const formAction={'tournament-form':'new-tournament','team-form':'new-team','score-form':'update-score'}[form.id];
    if (formAction && !canRoleAction(formAction)) { event.preventDefault(); toast('Your role cannot perform this action.'); return; }
    if (!token()) return;
    if (['tournament-form','team-form','score-form'].includes(form.id)) event.preventDefault();
    if (form.id==='tournament-form') {
      const name=form.elements.name.value.trim(), sport=form.elements.sport.value.split('·')[0].trim().toLowerCase(), starts=form.querySelector('#t-start').value;
      try { await request('/tournaments',{method:'POST',body:JSON.stringify({name,sport,season:'2025',starts_at:starts,dzongkhag:form.elements.dz.value})}); await refresh(); }
      catch(error){toast(`Could not save tournament: ${error.message}`);}
    } else if (form.id==='team-form') {
      const name=form.elements.name.value.trim(), dz=form.elements.dz.value, competition=form.querySelector('#team-tournament').value, tournament=tournaments.find(t=>t.name===competition);
      try { await request('/teams',{method:'POST',body:JSON.stringify({name,short_code:initials(name),dzongkhag:dz,...(tournament?.id?{tournament_id:tournament.id}:{})})}); await refresh(); }
      catch(error){toast(`Could not save team: ${error.message}`);}
    } else if (form.id==='score-form') {
      const match=fixtureData[Number(form.dataset.matchIndex)],matchID=safeUUID(match?.id); if (!matchID) {toast('This demo fixture is not saved in the database.');return;}
      const body=JSON.stringify({home:Number(form.elements.home.value),away:Number(form.elements.away.value)}),final=event.submitter?.value==='finalize';
      try { await request(`/matches/${encodeURIComponent(matchID)}/${final?'finalize':'live'}`,{method:final?'POST':'PATCH',body});await refresh(); }
      catch(error){toast(`Could not save score: ${error.message}`);}
    }
  });
  document.addEventListener('click',async(event)=>{
    if(event.target.closest('[data-action="generate-fixtures"]')) {
      if(!canRoleAction('generate-fixtures')) {toast('Only Super Admins can generate fixtures.');return;}
      if(!token()){toast('Sign in to generate saved fixtures.');return;}
      const tournament=tournaments.find(t=>t.id),tournamentID=safeUUID(tournament?.id);if(!tournamentID){toast('Create a tournament and register at least two teams first.');return;}
      try{const result=await request(`/tournaments/${encodeURIComponent(tournamentID)}/fixtures`,{method:'POST'});toast(`${result.fixtures_created} fixtures created.`);await refresh();}catch(error){toast(error.message);}
    }
  });
  document.querySelector('.top-avatar').classList.add('auth-button');
  // Clear legacy bearer tokens and restore the HttpOnly cookie session from the API.
  sessionStorage.removeItem('drukleague-session');
  void request('/auth/session').then(user => {
    const username=safeUsername(user?.username);
    sessionStorage.setItem(authKey,'true');
    sessionStorage.setItem(roleKey,safeRole(user?.role));
    sessionStorage.setItem(usernameKey,username);
    syncRoleIdentity();
    render(activePage);
    return refresh();
  }).catch(() => {
    sessionStorage.removeItem(authKey);
    sessionStorage.setItem(roleKey,'viewer');
    sessionStorage.removeItem(usernameKey);
    syncRoleIdentity();
    render(activePage);
  });
})();
