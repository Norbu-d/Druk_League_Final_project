(() => {
  let currentUser = null;
  const authKey = 'drukleague-authenticated';
  const roleKey = 'drukleague-role';
  const usernameKey = 'drukleague-username';
  const {currentRole, canRoleAction, roleIdentity, sanitizeRole:safeRole, sanitizeUsername:safeUsername, sanitizeUUID:safeUUID, safeAPIURL} = window.DrukLeaguePermissions;
  const token = () => Boolean(currentUser);
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
  function setFormBusy(form,busy){
    form.setAttribute('aria-busy',String(busy));
    for(const button of form.querySelectorAll('button[type="submit"]')){
      if(busy){button.dataset.originalLabel=button.textContent;button.disabled=true;button.textContent='Saving…'}
      else{button.disabled=false;if(button.dataset.originalLabel){button.textContent=button.dataset.originalLabel;delete button.dataset.originalLabel}}
    }
  }
  function showFormError(form,message){
    let alert=form.querySelector('[data-form-error]');
    if(!alert){alert=document.createElement('p');alert.dataset.formError='true';alert.setAttribute('role','alert');alert.className='form-error';form.querySelector('.modal-body')?.prepend(alert)}
    if(alert)alert.textContent=message;
  }
  // Saves a modal form: busy state while the request runs; on success close the
  // modal and confirm right away (the data refresh follows); on failure keep the
  // form open with the error inside it.
  async function saveForm(form, save, {success, failure}) {
    setFormBusy(form,true);
    let result;
    try { result=await save(); }
    catch(error){
      setFormBusy(form,false);
      showFormError(form,`${failure}: ${error.message}`);
      toast(`${failure}: ${error.message}`,'error');
      return;
    }
    document.getElementById('modal-root').innerHTML='';
    toast(typeof success==='function'?success(result):success,'success');
    await refresh();
  }
  const standingsState = {status:'signed-out', tournamentID:'', tournaments:[], rows:[], message:''};
  window.DRUKLEAGUE_STANDINGS_STATE = standingsState;
  async function renew() {
    const safeURL = safeAPIURL(apiRoot, '/auth/refresh', 'POST');
    if (!safeURL) return false;
    const response = await fetch(safeURL, {method:'POST', credentials:'same-origin', headers:headers()});
    if (!response.ok) { currentUser = null; window.DRUKLEAGUE_AUTHENTICATED = false; return false; }
    return true;
  }
  async function request(path, options={}, retried=false) {
    const safeURL = safeAPIURL(apiRoot, path, options.method);
    if (!safeURL) throw new Error('Invalid API request path.');
    const response = await fetch(safeURL, {...options, credentials:'same-origin', headers:{...headers(), ...(options.headers||{})}});
    if (response.status === 401 && !retried && !path.startsWith('/auth/') && await renew()) return request(path, options, true);
    const data = await response.json().catch(()=>({}));
    if (!response.ok) throw new Error(data.error || `Request failed (${response.status})`);
    return data;
  }
  const initials = (name) => name.split(/\s+/).map(part=>part[0]||'').join('').slice(0,2).toUpperCase();
  async function loadStandings(tournamentID, {renderPage=true}={}) {
    const safeID = safeUUID(tournamentID);
    standingsState.tournamentID = safeID || '';
    standingsState.rows = [];
    standingsState.message = '';
    if (!token()) {
      standingsState.status = 'signed-out';
    } else if (!standingsState.tournaments.length) {
      standingsState.status = 'no-tournaments';
    } else if (!safeID) {
      standingsState.status = 'error';
      standingsState.message = 'Choose a valid tournament to load standings.';
    } else {
      standingsState.status = 'loading';
      if (renderPage && activePage === 'standings') render(activePage);
      try {
        const response = await request(`/standings?tournament_id=${encodeURIComponent(safeID)}`);
        standingsState.rows = (Array.isArray(response) ? response : []).map(row => ({
          team_id: safeUUID(row.team_id) || '',
          name: typeof row.team === 'string' && row.team.trim() ? row.team.trim() : 'Unnamed team',
          dz: typeof row.dzongkhag === 'string' && row.dzongkhag.trim() ? row.dzongkhag.trim() : '—',
          played: Number(row.played) || 0,
          w: Number(row.won) || 0,
          d: Number(row.drawn) || 0,
          l: Number(row.lost) || 0,
          gf: Number(row.goals_for) || 0,
          ga: Number(row.goals_against) || 0,
          gd: Number(row.goal_difference) || 0,
          pts: Number(row.points) || 0
        }));
        standingsState.status = 'loaded';
        standingsState.updatedAt = new Date().toISOString();
      } catch (error) {
        standingsState.status = 'error';
        standingsState.message = error.message || 'Could not load standings.';
        window.DRUKLEAGUE_DASHBOARD_STATE = {status:'error',message:standingsState.message};
      }
    }
    if (renderPage && activePage === 'standings') render(activePage);
    return standingsState;
  }
  window.DRUKLEAGUE_LOAD_STANDINGS = loadStandings;
  function teamRecord(t, row, tournamentByID) {
    return {id:t.id,name:t.name,short:t.short_code||initials(t.name),color:'',dz:t.dzongkhag,tournamentID:t.tournament_id||'',tournamentName:tournamentByID.get(t.tournament_id)?.name||'Not registered to a tournament',played:row.played||0,w:row.won||0,d:row.drawn||0,l:row.lost||0,gf:row.goals_for||0,ga:row.goals_against||0,gd:row.goal_difference===undefined?'0':String(row.goal_difference),pts:row.points||0,form:[]};
  }
  function tournamentStatus(status) {
    if(status==='completed') return 'Completed';
    if(status==='in_progress') return 'In progress';
    return 'Registration open';
  }
  function tournamentRecord(t, teamCountByTournament, fixtureCountByTournament) {
    return {id:t.id,name:t.name,sport:t.sport,season:t.season,status:tournamentStatus(t.status),teams:teamCountByTournament.get(t.id)||0,fixtures:fixtureCountByTournament.get(t.id)||0,fill:0,theme:'',description:`${t.sport} · ${t.season}`,venue:t.venue,dz:t.dzongkhag};
  }
  function fixtureRecord(m, tournamentByID) {
    const kickoff=m.kickoff?new Date(m.kickoff):null;
    const hasKickoff=kickoff&&!Number.isNaN(kickoff.valueOf());
    let score='';
    if(m.result_home!==null) score=`${m.result_home} — ${m.result_away}`;
    else if(m.live_home!==null) score=`${m.live_home} — ${m.live_away}`;
    let status='UPCOMING';
    if(m.status==='completed') status='COMPLETED';
    else if(m.status==='live') status='LIVE';
    let matchResult=null;
    if(m.status==='completed') matchResult={homeGoals:m.result_home,awayGoals:m.result_away};
    return {id:m.id,tournamentId:m.tournament_id,tournamentName:tournamentByID.get(m.tournament_id)?.name||'',kickoff:m.kickoff||'',dateMonth:hasKickoff?new Intl.DateTimeFormat(undefined,{month:'short'}).format(kickoff).toUpperCase():'—',dateDay:hasKickoff?new Intl.DateTimeFormat(undefined,{day:'2-digit'}).format(kickoff):'—',dateWeekday:hasKickoff?new Intl.DateTimeFormat(undefined,{weekday:'short'}).format(kickoff).toUpperCase():'TBD',time:hasKickoff?kickoff.toLocaleTimeString([],{hour:'2-digit',minute:'2-digit'}):'TBD',period:m.status.toUpperCase(),home:m.home,h:initials(m.home),away:m.away,a:initials(m.away),aColor:'warm',venue:m.venue,status,score,official:'Assigned official',matchResult};
  }
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
    window.DRUKLEAGUE_DASHBOARD_STATE = {status:'loading'};
    standingsState.status = 'loading';
    if (['dashboard','standings','teams','matches','tournaments'].includes(activePage)) render(activePage);
    try {
      const [remoteTeams, remoteTournaments, remoteMatches] = await Promise.all([request('/teams'),request('/tournaments'),request('/matches')]);
      if (![remoteTeams,remoteTournaments,remoteMatches].every(Array.isArray)) throw new Error('The API returned an unexpected record list.');
      const standingsTournamentID=safeUUID(remoteTournaments.find(t=>t.id===standingsState.tournamentID)?.id || remoteTournaments[0]?.id);
      const seasons=remoteTournaments.map(t=>String(t.season||'')).filter(Boolean).sort((a,b)=>b.localeCompare(a,undefined,{numeric:true}));
      const activeSeason=seasons[0]||String(new Date().getFullYear());
      window.DRUKLEAGUE_ACTIVE_SEASON=activeSeason;
      const seasonButton=document.getElementById('season-button');
      if(seasonButton) seasonButton.innerHTML=`<span class="season-dot"></span> ${activeSeason} season <span aria-hidden="true">⌄</span>`;
      standingsState.tournaments = remoteTournaments.map(t=>({id:safeUUID(t.id),name:t.name,season:t.season})).filter(t=>t.id);
      standingsState.status = standingsState.tournaments.length ? 'loading' : 'no-tournaments';
      const tableState=await loadStandings(standingsTournamentID, {renderPage:false});
      const table=tableState.status==='loaded'?tableState.rows.map(row=>({team_id:row.team_id,played:row.played,won:row.w,drawn:row.d,lost:row.l,goals_for:row.gf,goals_against:row.ga,goal_difference:row.gd,points:row.pts})):[];
      const byTeam=new Map(table.map(row=>[row.team_id,row]));
      const tournamentByID=new Map(remoteTournaments.map(t=>[t.id,t]));
      const teamCountByTournament=new Map(),fixtureCountByTournament=new Map();
      for(const team of remoteTeams) if(team.tournament_id) teamCountByTournament.set(team.tournament_id,(teamCountByTournament.get(team.tournament_id)||0)+1);
      for(const match of remoteMatches) if(match.tournament_id) fixtureCountByTournament.set(match.tournament_id,(fixtureCountByTournament.get(match.tournament_id)||0)+1);
      teams.splice(0,teams.length,...remoteTeams.map(t=>teamRecord(t,byTeam.get(t.id)||{},tournamentByID)));
      tournaments.splice(0,tournaments.length,...remoteTournaments.map(t=>tournamentRecord(t,teamCountByTournament,fixtureCountByTournament)));
      fixtureData.splice(0,fixtureData.length,...remoteMatches.map(m=>fixtureRecord(m,tournamentByID)));
      window.DRUKLEAGUE_DASHBOARD_STATE = {status:'loaded'};
      render(activePage);
    } catch (error) {
      window.DRUKLEAGUE_DASHBOARD_STATE = {status:'error',message:error.message || 'Could not load saved project data.'};
      standingsState.status = 'error';
      standingsState.message = error.message || 'Could not load saved project data.';
    if (['dashboard','standings','teams','matches','tournaments'].includes(activePage)) render(activePage);
      toast(`Could not load saved project data: ${error.message}`,'error');
    }
  }
  async function showLogin() {
    if (token()) {
      try { await request('/auth/logout',{method:'POST'}); }
      catch (error) { toast(`Could not sign out: ${error.message}`,'error'); return; }
      currentUser = null;
      window.DRUKLEAGUE_DASHBOARD_STATE = {status:'signed-out'};
      sessionStorage.removeItem(authKey);
      sessionStorage.removeItem(roleKey);
      sessionStorage.removeItem(usernameKey);
      window.DRUKLEAGUE_AUTHENTICATED = false;
      syncRoleIdentity();
      render('dashboard');
      toast('Signed out.','success'); return;
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
      try {
        const result=await request('/auth/login',{method:'POST',body:JSON.stringify({username:form.elements.username.value,password:form.elements.password.value})});
        currentUser=result.user||null;
        window.DRUKLEAGUE_DASHBOARD_STATE = {status:'loading'};
        const username=safeUsername(currentUser?.username);
        sessionStorage.setItem(authKey,'true');
        sessionStorage.setItem(roleKey,safeRole(currentUser?.role));
        sessionStorage.setItem(usernameKey,username);
        window.DRUKLEAGUE_AUTHENTICATED=Boolean(currentUser);
        syncRoleIdentity();
        document.getElementById('modal-root').innerHTML='';
        render(activePage);
        toast(`Signed in as ${username||'user'}.`,'success');
        await refresh();
      }
      catch(error){
        toast(`Could not sign in: ${error.message}`,'error');
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
    if (formAction && !canRoleAction(formAction)) { event.preventDefault(); toast('Your role cannot perform this action.','error'); return; }
    if (!token()) return;
    if (['tournament-form','team-form','score-form'].includes(form.id)) event.preventDefault();
    if (form.id==='tournament-form') {
      const name=form.elements.name.value.trim(), sport=form.elements.sport.value.split('·')[0].trim().toLowerCase(), starts=form.querySelector('#t-start').value, season=form.elements.season.value;
      if(!name){showFormError(form,'Enter a tournament name.');return;}
      await saveForm(form,()=>request('/tournaments',{method:'POST',body:JSON.stringify({name,sport,season,starts_at:starts,dzongkhag:form.elements.dz.value})}),
        {success:`${name} created for the ${season} season. Teams can now register.`,failure:'Could not save tournament'});
    } else if (form.id==='team-form') {
      const name=form.elements.name.value.trim(), dz=form.elements.dz.value, tournamentID=safeUUID(form.elements.tournament_id.value), tournament=tournaments.find(t=>t.id===tournamentID);
      if(!name){showFormError(form,'Enter a team name.');return;}
      if(!tournament){showFormError(form,'Choose a valid tournament before registering the team.');return;}
      await saveForm(form,()=>request('/teams',{method:'POST',body:JSON.stringify({name,short_code:initials(name),dzongkhag:dz,tournament_id:tournament.id})}),
        {success:`${name} registered for ${tournament.name}.`,failure:'Could not register team'});
    } else if (form.id==='score-form') {
      const match=fixtureData[Number(form.dataset.matchIndex)],matchID=safeUUID(match?.id); if (!matchID) {showFormError(form,'This fixture is not saved in the database, so its score cannot be saved.');return;}
      const home=Number(form.elements.home.value),away=Number(form.elements.away.value),final=event.submitter?.value==='finalize';
      if(![home,away].every(goals=>Number.isInteger(goals)&&goals>=0&&goals<=99)){showFormError(form,'Scores must be whole numbers from 0 to 99.');return;}
      const scoreline=`${match.home} ${home}–${away} ${match.away}`;
      await saveForm(form,()=>request(`/matches/${encodeURIComponent(matchID)}/${final?'finalize':'live'}`,{method:final?'POST':'PATCH',body:JSON.stringify({home,away})}),
        {success:final?`Result finalized: ${scoreline}. Standings update from this result.`:`Live score saved: ${scoreline}. The result is still open.`,failure:final?'Could not finalize result':'Could not save live score'});
    }
  });
  document.addEventListener('click',async(event)=>{
    if(event.target.closest('[data-action="refresh-dashboard"]')) { await refresh(); return; }
    const generateButton=event.target.closest('[data-action="generate-fixtures"]');
    if(generateButton) {
      if(!canRoleAction('generate-fixtures')) {toast('Only Super Admins can generate fixtures.','error');return;}
      if(!token()){toast('Sign in to generate saved fixtures.');return;}
      if(generateButton.disabled) return;
      const tournament=tournaments.find(t=>t.id===window.DRUKLEAGUE_FIXTURES_TOURNAMENT_ID)||tournaments.find(t=>t.id),tournamentID=safeUUID(tournament?.id);if(!tournamentID){toast('Create a tournament and register at least two teams first.');return;}
      const label=generateButton.textContent;generateButton.disabled=true;generateButton.textContent='Generating…';
      let result;
      try{result=await request(`/tournaments/${encodeURIComponent(tournamentID)}/fixtures`,{method:'POST'});}
      catch(error){toast(`Could not generate fixtures: ${error.message}`,'error');return;}
      finally{generateButton.disabled=false;generateButton.textContent=label;}
      const created=Number(result?.fixtures_created)||0;
      toast(created?`${created} fixture${created===1?'':'s'} created for ${tournament.name}.`:`No new fixtures were created for ${tournament.name}. It may already have fixtures, or need at least two registered teams.`,created?'success':'info');
      await refresh();
    }
  });
  document.addEventListener('change', event => {
    if (event.target.id === 'standings-tournament') void loadStandings(event.target.value);
    if (event.target.id === 'fixtures-tournament') { window.DRUKLEAGUE_FIXTURES_TOURNAMENT_ID=event.target.value||'all'; render('matches'); }
    if (event.target.id === 'tournament-season-filter') { tournamentSeasonFilter=event.target.value||'all'; render('tournaments'); }
  });
  document.querySelector('.top-avatar').classList.add('auth-button');
  window.DRUKLEAGUE_AUTHENTICATED = false;
  sessionStorage.removeItem('drukleague-session');
  void (async () => {
    try {
      currentUser = await request('/auth/session');
    } catch (_) {
      if (await renew()) {
        try { currentUser = await request('/auth/session'); }
        catch (_) { currentUser = null; }
      }
    }
    if (currentUser) {
      sessionStorage.setItem(authKey, 'true');
      sessionStorage.setItem(roleKey, safeRole(currentUser.role));
      sessionStorage.setItem(usernameKey, safeUsername(currentUser.username));
    } else {
      sessionStorage.removeItem(authKey);
      sessionStorage.setItem(roleKey, 'viewer');
      sessionStorage.removeItem(usernameKey);
    }
    window.DRUKLEAGUE_AUTHENTICATED = Boolean(currentUser);
    window.DRUKLEAGUE_DASHBOARD_STATE = currentUser ? {status:'loading'} : {status:'signed-out'};
    syncRoleIdentity();
    render(activePage);
    if (currentUser) await refresh();
  })();
})();
