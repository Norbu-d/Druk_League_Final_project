// DrukLeague data layer: session, loading and saving through the BFF (/api).
// The session lives in HttpOnly cookies set by the BFF; this file only keeps the signed-in
// user's id, name and role for display. Every request is checked by the server again.
(() => {
  const { sanitizeRole, sanitizeUsername, sanitizeUUID, safeAPIURL } = window.DrukLeaguePermissions;
  const { store, ui, render, refreshView, toast } = window.DrukApp;
  const roleKey = 'drukleague-role';
  const usernameKey = 'drukleague-username';

  const defaultRoot = location.protocol === 'file:' ? 'http://127.0.0.1:8080/api' : `${location.origin}/api`;
  const apiRoot = (() => {
    try {
      const candidate = new URL(window.DRUKLEAGUE_API_BASE || defaultRoot, location.href);
      const sameOrigin = candidate.origin === location.origin && location.origin !== 'null';
      const localDev = ['http://127.0.0.1:8080', 'http://localhost:8080'].includes(candidate.origin);
      if ((!sameOrigin && !localDev) || candidate.pathname.replace(/\/+$/, '') !== '/api' || candidate.search || candidate.hash) return defaultRoot;
      return candidate.href.replace(/\/+$/, '');
    } catch (_) { return defaultRoot; }
  })();

  /* ------------------------------------------------------------ requests */
  async function renew() {
    const url = safeAPIURL(apiRoot, '/auth/refresh', 'POST');
    if (!url) return false;
    try {
      const response = await fetch(url, { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' } });
      return response.ok;
    } catch (_) { return false; }
  }

  async function request(path, { method = 'GET', body } = {}, retried = false) {
    const url = safeAPIURL(apiRoot, path, method);
    if (!url) throw new Error('Blocked an unexpected API request.');
    let response;
    try {
      response = await fetch(url, { method, credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: body === undefined ? undefined : JSON.stringify(body) });
    } catch (_) {
      throw new Error('Could not reach the DrukLeague API. Check that the services are running.');
    }
    // An expired access token is renewed once with the refresh cookie, then the request is retried.
    if (response.status === 401 && !retried && !path.startsWith('/auth/') && await renew()) return request(path, { method, body }, true);
    const data = await response.json().catch(() => ({}));
    if (response.status === 401 && !path.startsWith('/auth/') && !path.startsWith('/public/')) {
      setSignedOut();
      refreshView();
      void loadPublic();
      throw new Error('Your session has ended. Please sign in again.');
    }
    if (!response.ok) throw new Error(data.error || `Request failed (${response.status})`);
    return data;
  }

  /* ------------------------------------------------------------ normalising */
  const text = (v) => (typeof v === 'string' ? v.trim() : '');
  const num = (v) => (v === null || v === undefined || v === '' ? null : Number(v));
  const date = (v) => { const d = v ? new Date(v) : null; return d && !Number.isNaN(d.valueOf()) ? d : null; };
  const initials = (name) => name.split(/\s+/).map((w) => w[0] || '').join('').slice(0, 3).toUpperCase();

  const toTournament = (t) => ({ id: sanitizeUUID(t.id), name: text(t.name) || 'Unnamed tournament', sport: text(t.sport), season: text(String(t.season ?? '')), status: text(t.status) || 'registration_open', startsAt: date(t.starts_at), dzongkhag: text(t.dzongkhag), venue: text(t.venue) });
  const toTeam = (t) => ({ id: sanitizeUUID(t.id), name: text(t.name) || 'Unnamed team', short: text(t.short_code) || initials(text(t.name) || '?'), dzongkhag: text(t.dzongkhag), tournamentId: sanitizeUUID(t.tournament_id), managerId: sanitizeUUID(t.manager_id) });
  const toUser = (u) => ({ id: sanitizeUUID(u.id || u.sub), username: sanitizeUsername(u.username) || 'user', role: sanitizeRole(u.role) });
  function toMatch(m) {
    const status = ['scheduled', 'live', 'completed'].includes(m.status) ? m.status : 'scheduled';
    return {
      id: sanitizeUUID(m.id), tournamentId: sanitizeUUID(m.tournament_id), homeId: sanitizeUUID(m.home_team_id), awayId: sanitizeUUID(m.away_team_id),
      home: text(m.home) || 'Home team', away: text(m.away) || 'Away team', venue: text(m.venue), kickoff: date(m.kickoff), status,
      officialId: sanitizeUUID(m.official_id), liveHome: num(m.live_home), liveAway: num(m.live_away), resultHome: num(m.result_home), resultAway: num(m.result_away)
    };
  }
  const toRegistration = (r) => ({ id: sanitizeUUID(r.id), tournamentId: sanitizeUUID(r.tournament_id), teamId: sanitizeUUID(r.team_id), status: ['pending', 'approved', 'rejected'].includes(r.status) ? r.status : 'pending', submittedAt: date(r.submitted_at) });
  const toPlayer = (p) => ({ id: sanitizeUUID(p.id), teamId: sanitizeUUID(p.team_id), given: text(p.given_name), family: text(p.family_name), shirt: text(p.shirt_number) });
  const toVenue = (v) => ({ id: sanitizeUUID(v.id), name: text(v.name) || 'Unnamed venue', dzongkhag: text(v.dzongkhag), capacity: Number(v.capacity) || 0 });
  const toStanding = (row) => ({
    teamId: sanitizeUUID(row.team_id), name: text(row.team) || 'Unnamed team', dz: text(row.dzongkhag) || '—',
    played: Number(row.played) || 0, w: Number(row.won) || 0, d: Number(row.drawn) || 0, l: Number(row.lost) || 0,
    gf: Number(row.goals_for) || 0, ga: Number(row.goals_against) || 0, gd: Number(row.goal_difference) || 0, pts: Number(row.points) || 0
  });

  /* ------------------------------------------------------------ session */
  function setSignedIn(user) {
    const clean = toUser(user);
    store.session = { status: 'signed-in', user: clean };
    sessionStorage.setItem(roleKey, clean.role);
    sessionStorage.setItem(usernameKey, clean.username);
    window.DRUKLEAGUE_AUTHENTICATED = true;
    ui.matchTab = clean.role === 'match_official' ? 'mine' : 'all';
    ui.teamScope = clean.role === 'team_manager' ? 'mine' : 'all';
    return clean;
  }
  function setSignedOut() {
    store.session = { status: 'signed-out', user: null };
    store.data = { status: 'idle', message: '' };
    store.tournaments = []; store.teams = []; store.matches = []; store.users = []; store.usersStatus = 'idle';
    store.registrations = []; store.registrationsStatus = 'idle'; store.venues = []; store.venuesStatus = 'idle'; store.players = {};
    store.standings = { tournamentId: '', status: 'idle', rows: [], message: '', updatedAt: '' };
    sessionStorage.setItem(roleKey, 'viewer');
    sessionStorage.removeItem(usernameKey);
    window.DRUKLEAGUE_AUTHENTICATED = false;
  }

  async function signIn(username, password) {
    const result = await request('/auth/login', { method: 'POST', body: { username, password } });
    const user = setSignedIn(result.user || {});
    void reload();
    return user;
  }

  // Creates a Viewer account (the server fixes the role) and signs it in.
  async function signUp(username, password) {
    const result = await request('/auth/signup', { method: 'POST', body: { username, password } });
    if (!result.signed_in) throw new Error('Your account was created. Sign in with your new username and password.');
    const user = setSignedIn(result.user || {});
    void reload();
    return user;
  }

  async function signOut() {
    try { await request('/auth/logout', { method: 'POST' }); } catch (error) { toast(`Could not sign out cleanly: ${error.message}`, 'error'); }
    setSignedOut();
    ui.page = 'dashboard';
    render('dashboard');
    toast('Signed out.', 'success');
    void loadPublic();
  }

  /* ------------------------------------------------------------ loading */
  async function loadUsers() {
    if (store.session.user?.role !== 'super_admin') { store.users = []; store.usersStatus = 'idle'; return; }
    store.usersStatus = 'loading';
    try {
      const users = await request('/users');
      store.users = (Array.isArray(users) ? users : []).map(toUser).filter((u) => u.id);
      store.usersStatus = 'loaded';
    } catch (_) {
      store.usersStatus = 'error';
    }
  }

  // A list that only some roles may read: the admin sees everything, a team manager their own.
  async function loadRegistrations() {
    if (!['super_admin', 'team_manager'].includes(store.session.user?.role)) { store.registrations = []; store.registrationsStatus = 'idle'; return; }
    store.registrationsStatus = 'loading';
    try {
      const rows = await request('/registrations');
      store.registrations = (Array.isArray(rows) ? rows : []).map(toRegistration).filter((r) => r.id);
      store.registrationsStatus = 'loaded';
    } catch (_) { store.registrationsStatus = 'error'; }
  }

  async function loadVenues() {
    if (store.session.user?.role !== 'super_admin') { store.venues = []; store.venuesStatus = 'idle'; return; }
    store.venuesStatus = 'loading';
    try {
      const rows = await request('/venues');
      store.venues = (Array.isArray(rows) ? rows : []).map(toVenue).filter((v) => v.id);
      store.venuesStatus = 'loaded';
    } catch (_) { store.venuesStatus = 'error'; }
  }

  // Squads load per team when a squad is opened.
  async function loadPlayers(teamId) {
    const team = sanitizeUUID(teamId);
    if (!team || store.session.status !== 'signed-in') return;
    store.players[team] = { status: 'loading', rows: store.players[team]?.rows || [], message: '' };
    refreshView();
    try {
      const rows = await request(`/teams/${team}/players`);
      store.players[team] = { status: 'loaded', rows: (Array.isArray(rows) ? rows : []).map(toPlayer).filter((p) => p.id), message: '' };
    } catch (error) {
      store.players[team] = { status: 'error', rows: [], message: error.message };
    }
    refreshView();
  }

  async function loadStandings(tournamentId, { quiet = false } = {}) {
    const id = sanitizeUUID(tournamentId);
    const s = store.standings;
    s.tournamentId = id;
    s.rows = [];
    s.message = '';
    if (!id) { s.status = 'idle'; if (!quiet) refreshView(); return; }
    s.status = 'loading';
    if (!quiet) refreshView();
    try {
      const base = store.session.status === 'signed-in' ? '/standings' : '/public/standings';
      const rows = await request(`${base}?tournament_id=${encodeURIComponent(id)}`);
      if (store.standings.tournamentId !== id) return; // a newer choice won
      s.rows = (Array.isArray(rows) ? rows : []).map(toStanding);
      s.status = 'loaded';
      s.updatedAt = new Date().toISOString();
    } catch (error) {
      s.status = 'error';
      s.message = error.message;
    }
    if (!quiet) refreshView();
  }

  let loading = null;
  // Loads tournaments, teams and matches (plus accounts for the admin). One failing list does
  // not hide the others; the page shows what loaded and a message names what did not.
  async function reload() {
    if (store.session.status !== 'signed-in') return;
    if (loading) return loading;
    loading = (async () => {
      if (store.data.status !== 'loaded') store.data = { status: 'loading', message: '' };
      refreshView();
      const [tournaments, teams, matches] = await Promise.allSettled([request('/tournaments'), request('/teams'), request('/matches'), loadUsers(), loadRegistrations(), loadVenues()]);
      if (store.session.status !== 'signed-in') return;
      const failed = [];
      if (tournaments.status === 'fulfilled' && Array.isArray(tournaments.value)) store.tournaments = tournaments.value.map(toTournament).filter((t) => t.id); else failed.push('tournaments');
      if (teams.status === 'fulfilled' && Array.isArray(teams.value)) store.teams = teams.value.map(toTeam).filter((t) => t.id); else failed.push('teams');
      if (matches.status === 'fulfilled' && Array.isArray(matches.value)) store.matches = matches.value.map(toMatch).filter((m) => m.id); else failed.push('matches');
      if (failed.length === 3) {
        store.data = { status: 'error', message: [tournaments, teams, matches].find((r) => r.status === 'rejected')?.reason?.message || 'The API did not return any data.' };
      } else {
        store.data = { status: 'loaded', message: '' };
        if (failed.length) toast(`Could not load ${failed.join(' and ')}. Other data is shown.`, 'error');
      }
      const keep = store.tournaments.some((t) => t.id === store.standings.tournamentId) ? store.standings.tournamentId : store.tournaments[0]?.id;
      await loadStandings(keep, { quiet: true });
      refreshView();
    })().finally(() => { loading = null; });
    return loading;
  }

  // Visitors who are not signed in see the public, read-only fixtures, results and standings.
  let loadingPublic = null;
  async function loadPublic() {
    if (store.session.status === 'signed-in') return;
    if (loadingPublic) return loadingPublic;
    loadingPublic = (async () => {
      store.data = { status: 'loading', message: '' };
      refreshView();
      const [tournaments, teams, matches] = await Promise.allSettled([request('/public/tournaments'), request('/public/teams'), request('/public/matches')]);
      if (store.session.status === 'signed-in') return;
      if ([tournaments, teams, matches].every((r) => r.status === 'rejected')) {
        store.data = { status: 'error', message: tournaments.reason?.message || 'The API did not return any data.' };
        refreshView();
        return;
      }
      const list = (r, map) => (r.status === 'fulfilled' && Array.isArray(r.value) ? r.value.map(map).filter((x) => x.id) : []);
      store.tournaments = list(tournaments, toTournament);
      store.teams = list(teams, toTeam);
      store.matches = list(matches, toMatch);
      store.data = { status: 'loaded', message: '' };
      const keep = store.tournaments.some((t) => t.id === store.standings.tournamentId) ? store.standings.tournamentId : store.tournaments[0]?.id;
      await loadStandings(keep, { quiet: true });
      refreshView();
    })().finally(() => { loadingPublic = null; });
    return loadingPublic;
  }

  // Live scores: every 15 s, refresh the matches (and the open table) on the pages that show
  // scores. Skips hidden tabs, the landing page, open dialogs and focused form fields, so it
  // never redraws under someone who is typing.
  const LIVE_REFRESH_MS = 15000;
  const LIVE_PAGES = ['dashboard', 'matches', 'tournament', 'standings'];
  let refreshing = false;
  async function refreshLive() {
    if (refreshing || document.hidden || window.DrukLanding?.active?.() || store.data.status !== 'loaded' || !LIVE_PAGES.includes(ui.page)) return;
    if (document.querySelector('#modal-root .modal') || !document.getElementById('auth-screen').hidden) return;
    const focused = document.activeElement;
    if (focused && ['INPUT', 'SELECT', 'TEXTAREA'].includes(focused.tagName)) return;
    refreshing = true;
    try {
      const rows = await request(store.session.status === 'signed-in' ? '/matches' : '/public/matches');
      if (Array.isArray(rows)) store.matches = rows.map(toMatch).filter((m) => m.id);
      if (store.standings.tournamentId) await loadStandings(store.standings.tournamentId, { quiet: true });
      refreshView();
    } catch (_) {
      // A missed refresh is harmless; the next one tries again.
    } finally {
      refreshing = false;
    }
  }
  setInterval(refreshLive, LIVE_REFRESH_MS);

  /* ------------------------------------------------------------ saves */
  const id = (value, what) => { const clean = sanitizeUUID(value); if (!clean) throw new Error(`Choose a valid ${what}.`); return clean; };
  const api = {
    signIn, signUp, signOut, reload, loadStandings, loadPlayers,
    createTournament: (body) => request('/tournaments', { method: 'POST', body }),
    createTeam: (body) => request('/teams', { method: 'POST', body: { ...body, tournament_id: id(body.tournament_id, 'tournament') } }),
    generateFixtures: (tournamentId) => request(`/tournaments/${id(tournamentId, 'tournament')}/fixtures`, { method: 'POST' }),
    saveLive: (matchId, home, away) => request(`/matches/${id(matchId, 'match')}/live`, { method: 'PATCH', body: { home, away } }),
    finalize: (matchId, home, away) => request(`/matches/${id(matchId, 'match')}/finalize`, { method: 'POST', body: { home, away } }),
    assignOfficial: (matchId, officialId) => request(`/matches/${id(matchId, 'match')}/official`, { method: 'PATCH', body: { official_id: id(officialId, 'official') } }),
    createUser: (body) => request('/users', { method: 'POST', body }),
    registerTeam: (tournamentId, teamId) => request('/registrations', { method: 'POST', body: { tournament_id: id(tournamentId, 'tournament'), team_id: id(teamId, 'team') } }),
    reviewRegistration: (registrationId, approved, reason = '') => request(`/registrations/${id(registrationId, 'registration')}`, { method: 'PATCH', body: { approved: Boolean(approved), reason } }),
    createPlayer: (teamId, body) => request(`/teams/${id(teamId, 'team')}/players`, { method: 'POST', body }),
    updatePlayer: (playerId, body) => request(`/players/${id(playerId, 'player')}`, { method: 'PATCH', body }),
    createVenue: (body) => request('/venues', { method: 'POST', body }),
    createMatch: (body) => request('/matches', { method: 'POST', body: { ...body, tournament_id: id(body.tournament_id, 'tournament'), home_team_id: id(body.home_team_id, 'home team'), away_team_id: id(body.away_team_id, 'away team'), official_id: sanitizeUUID(body.official_id) } })
  };
  window.DrukAPI = Object.freeze(api);

  /* ------------------------------------------------------------ start-up: restore a saved session */
  sessionStorage.removeItem('drukleague-session');
  sessionStorage.removeItem('drukleague-authenticated');
  void (async () => {
    let user = null;
    try { user = await request('/auth/session'); } catch (_) {
      if (await renew()) { try { user = await request('/auth/session'); } catch (_) { user = null; } }
    }
    if (user && sanitizeUUID(user.sub)) setSignedIn(user); else setSignedOut();
    render(ui.page, { history: false, scroll: false });
    if (store.session.status === 'signed-in') await reload(); else await loadPublic();
  })();
})();
