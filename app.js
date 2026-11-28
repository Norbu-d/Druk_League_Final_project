// DrukLeague workspace: screens, navigation and forms.
// Data and sign-in live in api.js (window.DrukAPI); this file renders `store` and handles UI events.
// The server checks every role again; hiding a control here is presentation only.

/* ---------------------------------------------------------------- icons */
const ICONS = {
  grid: '<rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/>',
  trophy: '<path d="M8 21h8M12 17v4M7 4h10v5a5 5 0 0 1-10 0V4Z"/><path d="M7 6H4v2a4 4 0 0 0 4 4m9-6h3v2a4 4 0 0 1-4 4"/>',
  users: '<path d="M16 21v-2a4 4 0 0 0-4-4H7a4 4 0 0 0-4 4v2M9.5 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z"/><path d="M17 11a4 4 0 0 0 0-8m4 18v-2a4 4 0 0 0-3-3.87"/>',
  calendar: '<rect x="3" y="5" width="18" height="16" rx="2"/><path d="M16 3v4M8 3v4M3 10h18"/>',
  chart: '<path d="M3 3v18h18M8 15l4-4 4 3 5-7"/>',
  shield: '<path d="M12 22s8-4 8-11V5l-8-3-8 3v6c0 7 8 11 8 11Z"/><path d="m9 12 2 2 4-4"/>',
  pin: '<path d="M20 10c0 5-8 12-8 12S4 15 4 10a8 8 0 1 1 16 0Z"/><circle cx="12" cy="10" r="2.5"/>',
  file: '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8Z"/><path d="M14 2v6h6M8 13h8m-8 4h8"/>',
  search: '<circle cx="11" cy="11" r="7"/><path d="m20 20-4-4"/>',
  menu: '<path d="M4 6h16M4 12h16M4 18h16"/>',
  plus: '<path d="M12 5v14m-7-7h14"/>',
  download: '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4m4-5 5 5 5-5m-5 5V3"/>',
  key: '<circle cx="8" cy="15" r="4"/><path d="m10.8 12.2 9.2-9.2m-3 3 2 2m-5 1 2 2"/>',
  login: '<path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4M10 17l5-5-5-5M15 12H3"/>',
  logout: '<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9"/>',
  check: '<path d="m5 12 5 5 9-10"/>',
  whistle: '<circle cx="9" cy="14" r="5"/><path d="M13 11 21 7v4l-6 2M9 14h.01"/>',
  info: '<circle cx="12" cy="12" r="9"/><path d="M12 16v-4m0-4h.01"/>',
  alert: '<path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z"/><path d="M12 9v4m0 4h.01"/>',
  eye: '<path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Z"/><circle cx="12" cy="12" r="3"/>',
  lock: '<rect x="4" y="11" width="16" height="10" rx="2"/><path d="M8 11V7a4 4 0 0 1 8 0v4"/>',
  refresh: '<path d="M21 12a9 9 0 0 1-15.5 6.2M3 12A9 9 0 0 1 18.5 5.8M21 4v5h-5M3 20v-5h5"/>',
  home: '<path d="m3 11 9-8 9 8v9a1 1 0 0 1-1 1h-5v-6H9v6H4a1 1 0 0 1-1-1Z"/>',
  shirt: '<path d="M8 3 3 6l2 5 3-1v11h8V10l3 1 2-5-5-3a4 4 0 0 1-8 0Z"/>',
  flag: '<path d="M5 21V4m0 0h11l-2 4 2 4H5"/>',
  x: '<path d="M6 6l12 12M18 6 6 18"/>',
  edit: '<path d="M4 20h4L19 9l-4-4L4 16v4Z"/><path d="m13.5 6.5 4 4"/>',
  userplus: '<path d="M15 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M8.5 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM19 8v6m-3-3h6"/>'
};
const icon = (name) => `<svg class="i" viewBox="0 0 24 24" aria-hidden="true">${ICONS[name] || ICONS.grid}</svg>`;
document.querySelectorAll('[data-icon]').forEach((el) => { el.innerHTML = icon(el.dataset.icon); });

/* ---------------------------------------------------------------- state */
// Filled by api.js. Shapes are normalised there.
const store = {
  session: { status: 'checking', user: null },      // checking | signed-out | signed-in
  data: { status: 'idle', message: '' },            // idle | loading | loaded | error
  tournaments: [], teams: [], matches: [],
  users: [], usersStatus: 'idle',                   // admin only
  registrations: [], registrationsStatus: 'idle',   // admin: all; team manager: their own
  venues: [], venuesStatus: 'idle',                 // admin only
  players: {},                                      // team id -> { status, rows, message }
  standings: { tournamentId: '', status: 'idle', rows: [], message: '', updatedAt: '' }
};
const ui = { page: 'dashboard', query: '', matchTab: 'all', matchTournament: 'all', season: 'all', teamScope: 'all', userRole: 'all', squadTeam: '', regTab: 'pending', sport: 'all', tournamentId: '', tTab: 'matches' };

const { rolePages, currentRole, canRoleAction, roleLabels } = window.DrukLeaguePermissions;
const ROLE_INFO = {
  super_admin: { icon: 'key', title: 'Super Admin', can: ['Create tournaments, venues and fixtures', 'Approve team registrations', 'Create accounts and assign officials'] },
  match_official: { icon: 'whistle', title: 'Match Official', can: ['See the matches assigned to you', 'Update the live score during play', 'Finalise the result at full time'] },
  team_manager: { icon: 'users', title: 'Team Manager', can: ['Enter teams into open tournaments', 'Build your squad list', 'Follow approval, fixtures and results'] },
  viewer: { icon: 'eye', title: 'Viewer', can: ['Follow fixtures, results and standings', 'Browse teams and their squads', 'Download tables as CSV'] }
};
const DZONGKHAGS = ['Bumthang', 'Chhukha', 'Dagana', 'Gasa', 'Haa', 'Lhuentse', 'Mongar', 'Paro', 'Pema Gatshel', 'Punakha', 'Samdrup Jongkhar', 'Samtse', 'Sarpang', 'Thimphu', 'Trashigang', 'Trashiyangtse', 'Trongsa', 'Tsirang', 'Wangdue Phodrang', 'Zhemgang'];
const SPORTS = ['Football', 'Futsal', 'Basketball', 'Volleyball', 'Archery'];
const PAGE_LABELS = { dashboard: 'Overview', tournaments: 'Tournaments', tournament: 'Tournament', teams: 'Teams', squads: 'Squads', registrations: 'Registrations', matches: 'Matches & scores', standings: 'Standings', officials: 'Officials', users: 'Users & roles', venues: 'Venues', reports: 'Reports' };
// Pages a visitor can read without an account (served by the BFF's public routes).
const PUBLIC_PAGES = ['dashboard', 'tournaments', 'tournament', 'teams', 'matches', 'standings'];

/* ---------------------------------------------------------------- helpers */
function esc(value) {
  return String(value ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}
const initials = (name) => String(name || '?').split(/\s+/).map((w) => w[0] || '').join('').slice(0, 2).toUpperCase() || '?';
const me = () => store.session.user || { id: '', username: '', role: 'viewer' };
const role = () => (store.session.status === 'signed-in' ? currentRole() : 'viewer');
const signedIn = () => store.session.status === 'signed-in';
const loaded = () => store.data.status === 'loaded';
const plural = (n, one, many = `${one}s`) => `${n} ${n === 1 ? one : many}`;
const byId = (list, id) => list.find((item) => item.id === id);
const tournamentName = (id) => byId(store.tournaments, id)?.name || 'No tournament';
const userName = (id) => byId(store.users, id)?.username || '';
const officials = () => store.users.filter((u) => u.role === 'match_official');
const teamName = (id) => byId(store.teams, id)?.name || 'Unknown team';
// Registrations: fixture generation only uses approved entries.
const registrationFor = (teamId, tournamentId) => store.registrations.find((r) => r.teamId === teamId && r.tournamentId === tournamentId);
const approvedIn = (tournamentId) => store.registrations.filter((r) => r.tournamentId === tournamentId && r.status === 'approved');
const pendingRegistrations = () => store.registrations.filter((r) => r.status === 'pending');
const tracksRegistrations = () => ['super_admin', 'team_manager'].includes(role()) && store.registrationsStatus === 'loaded';
// Teams that play in a tournament: linked on the team record or entered through a registration.
function teamsIn(tournamentId) {
  const ids = new Set(store.registrations.filter((r) => r.tournamentId === tournamentId && r.status !== 'rejected').map((r) => r.teamId));
  return store.teams.filter((t) => t.tournamentId === tournamentId || ids.has(t.id));
}
const canEditSquad = (team) => Boolean(team) && (role() === 'super_admin' || (role() === 'team_manager' && team.managerId === me().id));
function registrationBadge(reg) {
  if (!reg) return '<span class="status status-none">Not entered</span>';
  if (reg.status === 'approved') return '<span class="status status-completed">Approved</span>';
  if (reg.status === 'rejected') return '<span class="status status-rejected">Rejected</span>';
  return '<span class="status status-pending">Waiting for approval</span>';
}

function dateParts(date) {
  if (!(date instanceof Date) || Number.isNaN(date.valueOf())) return { mon: 'TBD', day: '—', wk: '', time: 'Time TBD', long: 'Date to be confirmed' };
  return {
    mon: date.toLocaleDateString(undefined, { month: 'short' }),
    day: date.toLocaleDateString(undefined, { day: '2-digit' }),
    wk: date.toLocaleDateString(undefined, { weekday: 'short' }),
    time: date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    long: date.toLocaleDateString(undefined, { weekday: 'long', day: 'numeric', month: 'long' })
  };
}
const formatDate = (date) => (date instanceof Date && !Number.isNaN(date.valueOf()) ? date.toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' }) : 'Not set');

function matchScore(m) {
  if (m.status === 'completed' && m.resultHome != null) return [m.resultHome, m.resultAway];
  if (m.liveHome != null) return [m.liveHome, m.liveAway];
  return null;
}
const isMine = (m) => Boolean(m.officialId) && m.officialId === me().id;
const myTeams = () => store.teams.filter((t) => t.managerId && t.managerId === me().id);
function sortByKickoff(list) {
  return [...list].sort((a, b) => (a.kickoff?.valueOf() ?? Infinity) - (b.kickoff?.valueOf() ?? Infinity));
}
// Unfinished matches in the order people care about: live now, then upcoming kick-offs,
// then overdue fixtures that were never scored.
function nextUp(list) {
  const recent = Date.now() - 3 * 3600 * 1000;
  const open = list.filter((m) => m.status !== 'completed');
  const live = sortByKickoff(open.filter((m) => m.status === 'live'));
  const rest = open.filter((m) => m.status !== 'live');
  const future = sortByKickoff(rest.filter((m) => !m.kickoff || m.kickoff.valueOf() >= recent));
  const overdue = sortByKickoff(rest.filter((m) => m.kickoff && m.kickoff.valueOf() < recent));
  return [...live, ...future, ...overdue];
}
function teamStats(teamId) {
  const s = { p: 0, w: 0, d: 0, l: 0, gf: 0, ga: 0 };
  for (const m of store.matches) {
    if (m.status !== 'completed' || m.resultHome == null) continue;
    let mine; let theirs;
    if (m.homeId === teamId) { mine = m.resultHome; theirs = m.resultAway; } else if (m.awayId === teamId) { mine = m.resultAway; theirs = m.resultHome; } else continue;
    s.p += 1; s.gf += mine; s.ga += theirs;
    if (mine > theirs) s.w += 1; else if (mine < theirs) s.l += 1; else s.d += 1;
  }
  s.gd = s.gf - s.ga; s.pts = s.w * 3 + s.d;
  return s;
}
const COVERS = ['night-match', 'stadium-dzong', 'bhutan-maldives', 'saff-final', 'hero-stadium', 'scoreboard', 'night-shot', 'sunset-save'];
function coverFor(t) {
  const sport = String(t.sport || '').toLowerCase();
  if (sport.includes('volley')) return 'volleyball';
  if (sport.includes('arch')) return 'archery-aim';
  let h = 0;
  for (const c of String(t.id || t.name)) h = (h * 31 + c.charCodeAt(0)) >>> 0;
  return COVERS[h % COVERS.length];
}
const titleCase = (s) => String(s || '').replace(/(^|\s)\S/g, (c) => c.toUpperCase());

/* ---------------------------------------------------------------- sports */
// Tournaments are filtered and grouped by sport so a football fan never has to dig through archery.
const SPORT_INFO = { football: { label: 'Football', emoji: '⚽' }, futsal: { label: 'Futsal', emoji: '🥅' }, basketball: { label: 'Basketball', emoji: '🏀' }, volleyball: { label: 'Volleyball', emoji: '🏐' }, archery: { label: 'Archery', emoji: '🏹' } };
const sportKey = (s) => String(s || '').toLowerCase().trim();
// Mirror of the server's sport rules (packages/servicekit/core/sports.go) for labels and early
// feedback; the Score service enforces them.
const SPORT_RULES = {
  football: { unit: 'Goals', cols: ['GF', 'GA', 'GD'], words: ['goals for', 'goals against', 'goal difference'], max: 99, draw: true, points: '3 points for a win, 1 for a draw' },
  futsal: { unit: 'Goals', cols: ['GF', 'GA', 'GD'], words: ['goals for', 'goals against', 'goal difference'], max: 99, draw: true, points: '3 points for a win, 1 for a draw' },
  basketball: { unit: 'Points', cols: ['PF', 'PA', '+/−'], words: ['points for', 'points against', 'point difference'], max: 250, draw: false, quick: [2, 3], points: '2 points for a win, 1 for a loss; no draws' },
  volleyball: { unit: 'Sets', cols: ['SW', 'SL', 'SD'], words: ['sets won', 'sets lost', 'set difference'], max: 3, draw: false, bestOf: 5, points: '3 points for a 3–0 or 3–1 win, 2 for 3–2 (the loser gets 1)' },
  archery: { unit: 'Points', cols: ['PF', 'PA', 'PD'], words: ['points for', 'points against', 'point difference'], max: 99, draw: false, points: '2 points for a win; no draws' }
};
const rulesFor = (sport) => SPORT_RULES[sportKey(sport)] || SPORT_RULES.football;
// Same checks as the Score service, so the official sees the reason before saving.
function scoreProblem(sport, home, away, final) {
  const r = rulesFor(sport);
  if (![home, away].every((n) => Number.isInteger(n) && n >= 0 && n <= r.max)) return `${r.unit} must be whole numbers from 0 to ${r.max}.`;
  if (!final) return '';
  if (!r.draw && home === away) return `A ${sportInfo(sport).label.toLowerCase()} match cannot end in a draw.`;
  if (r.bestOf) { const need = Math.floor(r.bestOf / 2) + 1; if (Math.max(home, away) !== need || Math.min(home, away) >= need) return `A best-of-${r.bestOf} match ends when one side wins ${need} sets.`; }
  return '';
}
const standingsLegend = (sport) => { const r = rulesFor(sport); return `P played · W won${r.draw ? ' · D drawn' : ''} · L lost · ${r.cols.map((c, i) => `${c} ${r.words[i]}`).join(' · ')} · Pts points (${r.points})`; };
const sportInfo = (s) => SPORT_INFO[sportKey(s)] || { label: titleCase(s) || 'Other', emoji: '🏅' };
const sportOfMatch = (m) => byId(store.tournaments, m.tournamentId)?.sport || '';
const inSport = (sport) => ui.sport === 'all' || sportKey(sport) === ui.sport;
const inSportT = (t) => inSport(t.sport);
const inSportM = (m) => inSport(sportOfMatch(m));
const sportChip = (sport) => `<span class="sport-chip sport-${esc(sportKey(sport) || 'other')}">${sportInfo(sport).emoji} ${esc(sportInfo(sport).label)}</span>`;
function sportsPresent() {
  const order = Object.keys(SPORT_INFO);
  const rank = (k) => (order.includes(k) ? order.indexOf(k) : 99);
  return [...new Set(store.tournaments.map((t) => sportKey(t.sport)).filter(Boolean))].sort((a, b) => rank(a) - rank(b));
}
// Sport pills: "All sports" plus one pill per sport that has tournaments.
function sportFilter() {
  const present = sportsPresent();
  if (present.length < 2) return '';
  if (ui.sport !== 'all' && !present.includes(ui.sport)) ui.sport = 'all';
  const pills = [['all', '🏆', 'All sports', store.tournaments.length], ...present.map((k) => [k, sportInfo(k).emoji, sportInfo(k).label, store.tournaments.filter((t) => sportKey(t.sport) === k).length])];
  return `<div class="sport-filter" role="group" aria-label="Filter by sport">${pills.map(([k, e, l, n]) => `<button type="button" class="sport-pill ${ui.sport === k ? 'active' : ''}" data-action="sport" data-sport="${esc(k)}" aria-pressed="${ui.sport === k}"><span aria-hidden="true">${e}</span>${esc(l)}<span class="count">${n}</span></button>`).join('')}</div>`;
}
const staffView = () => signedIn() && ['super_admin', 'match_official'].includes(role());
// One tournament card. Fans click anywhere on it; staff get explicit buttons so admin actions still work.
function tournamentCard(t) {
  const matches = store.matches.filter((m) => m.tournamentId === t.id);
  const live = matches.filter((m) => m.status === 'live').length;
  const played = matches.filter((m) => m.status === 'completed').length;
  const fan = !signedIn() || role() === 'viewer';
  const actions = fan ? '' : [
    `<button type="button" class="btn btn-sm btn-primary" data-page="tournament" data-tournament="${esc(t.id)}">Open</button>`,
    button('Generate fixtures', 'generate', { cls: 'btn-sm', data: { tournament: t.id }, roleAction: 'generate-fixtures' }),
    role() === 'team_manager' && t.status === 'registration_open' ? button('Enter a team', 'new-team', { cls: 'btn-sm', data: { tournament: t.id } }) : ''
  ].join('');
  const entries = role() === 'super_admin' && tracksRegistrations() ? `<small class="t-entries">${approvedIn(t.id).length} approved · ${store.registrations.filter((r) => r.tournamentId === t.id && r.status === 'pending').length} waiting</small>` : '';
  return `<article class="t-card ${fan ? 't-card-link' : ''}"${fan ? ` data-page="tournament" data-tournament="${esc(t.id)}" tabindex="0" role="link" aria-label="Open ${esc(t.name)}"` : ''}><div class="t-cover"><img src="./assets/landing/${coverFor(t)}-sm.jpg" alt="" loading="lazy" decoding="async" />${sportChip(t.sport)}${live ? '<span class="live-tag"><span class="live-dot"></span>LIVE</span>' : ''}<h3>${esc(t.name)}</h3></div>
    <div class="t-body"><div class="t-stats"><div><b>${teamsIn(t.id).length}</b><small>Teams</small></div><div><b>${matches.length}</b><small>Matches</small></div><div><b>${played}</b><small>Played</small></div></div>
    <div class="t-meta"><span>${icon('pin')} ${esc(t.dzongkhag || 'Dzongkhag not set')} · ${esc(formatDate(t.startsAt))}</span>${tournamentStatus(t)}</div>${entries}
    ${actions ? `<div class="t-actions">${actions}</div>` : '<div class="t-open">See matches and table →</div>'}</div></article>`;
}

/* ---------------------------------------------------------------- building blocks */
function pageHead(eyebrow, title, desc, actions = '') {
  return `<header class="page-head"><div><div class="eyebrow">${esc(eyebrow)}</div><h1>${title}</h1>${desc ? `<p>${esc(desc)}</p>` : ''}</div>${actions ? `<div class="page-actions">${actions}</div>` : ''}</header>`;
}
function button(label, action, { cls = '', ic = '', data = {}, roleAction = '' } = {}) {
  if (roleAction && !canRoleAction(roleAction)) return '';
  const attrs = Object.entries(data).map(([k, v]) => ` data-${k}="${esc(v)}"`).join('');
  return `<button type="button" class="btn ${cls}" data-action="${action}"${attrs}>${ic ? icon(ic) : ''}${esc(label)}</button>`;
}
function metric(label, value, foot, ic) {
  return `<article class="metric"><div class="metric-top"><span>${esc(label)}</span><span class="icon">${icon(ic)}</span></div><b>${esc(value)}</b><small>${esc(foot)}</small></article>`;
}
function empty(title, text, { ic = 'info', action = '' } = {}) {
  return `<div class="empty"><span class="empty-icon">${icon(ic)}</span><strong>${esc(title)}</strong>${text ? `<p>${esc(text)}</p>` : ''}${action}</div>`;
}
function notice(text, warn = false) {
  return `<div class="notice ${warn ? 'notice-warn' : ''}">${icon(warn ? 'alert' : 'info')}<span>${text}</span></div>`;
}
function panel(title, sub, body, tools = '') {
  return `<section class="panel"><div class="panel-head"><div><h2>${esc(title)}</h2>${sub ? `<p>${esc(sub)}</p>` : ''}</div>${tools}</div><div class="panel-body">${body}</div></section>`;
}
function statusBadge(m) {
  if (m.status === 'completed') return '<span class="status status-completed">Final</span>';
  if (m.status === 'live') return '<span class="status status-live">Live</span>';
  return '<span class="status status-upcoming">Upcoming</span>';
}
function tournamentStatus(t) {
  if (t.status === 'completed') return '<span class="status status-completed">Completed</span>';
  if (t.status === 'in_progress') return '<span class="status status-live">In progress</span>';
  return '<span class="status status-open">Registration open</span>';
}
function officialLabel(m) {
  if (!m.officialId) return 'No official yet';
  if (isMine(m)) return 'You are the official';
  return userName(m.officialId) || 'Official assigned';
}
function roleBadge(r) { return `<span class="badge badge-${esc(r)}">${esc(roleLabels[r] || r)}</span>`; }

function matchCard(m, { compact = false } = {}) {
  const d = dateParts(m.kickoff);
  const score = matchScore(m);
  const actions = [];
  if (!compact) {
    if (role() === 'match_official' && isMine(m) && m.status !== 'completed') {
      actions.push(button(m.status === 'live' ? 'Update score' : 'Start scoring', 'score', { cls: 'btn-primary btn-sm', data: { id: m.id } }));
    }
    if (canRoleAction('assign-official') && m.status !== 'completed') {
      actions.push(button(m.officialId ? 'Change official' : 'Assign official', 'assign', { cls: 'btn-sm', ic: 'whistle', data: { id: m.id } }));
    }
  }
  return `<article class="match-card ${isMine(m) && role() === 'match_official' ? 'is-mine' : ''}">
    <div class="match-date"><small>${esc(d.mon)}</small><strong>${esc(d.day)}</strong><small>${esc(d.wk)}</small></div>
    <div class="match-teams">
      <div class="match-team"><span class="crest">${esc(initials(m.home))}</span><span>${esc(m.home)}</span><span class="score">${score ? esc(score[0]) : ''}</span></div>
      <div class="match-team"><span class="crest alt">${esc(initials(m.away))}</span><span>${esc(m.away)}</span><span class="score">${score ? esc(score[1]) : ''}</span></div>
    </div>
    <div class="match-meta">${statusBadge(m)}<span>${sportInfo(sportOfMatch(m)).emoji} ${esc(d.time)} · ${esc(m.venue || 'Venue TBD')}</span>${compact ? '' : `${ui.page === 'tournament' ? '' : `<button type="button" class="link-btn match-tournament" data-page="tournament" data-tournament="${esc(m.tournamentId)}">${esc(tournamentName(m.tournamentId))}</button>`}${staffView() ? `<span class="official">${icon('whistle')}${esc(officialLabel(m))}</span>` : ''}`}</div>
    ${actions.length ? `<div class="match-actions">${actions.join('')}</div>` : ''}
  </article>`;
}
// Column names follow the sport (goals, points or sets); the drawn column is hidden where draws cannot happen.
function standingsTable(rows, limit = 0, sport = '') {
  const r = rulesFor(sport);
  const ranked = [...rows].sort((a, b) => b.pts - a.pts || b.gd - a.gd || b.gf - a.gf || a.name.localeCompare(b.name));
  const shown = limit ? ranked.slice(0, limit) : ranked;
  const th = (c, title) => `<th class="num" title="${esc(title)}">${esc(c)}</th>`;
  return `<div class="table-wrap"><table class="table"><thead><tr><th>#</th><th>Team</th>${th('P', 'played')}${th('W', 'won')}${r.draw ? th('D', 'drawn') : ''}${th('L', 'lost')}${limit ? '' : th(r.cols[0], r.words[0]) + th(r.cols[1], r.words[1])}${th(r.cols[2], r.words[2])}${th('Pts', 'points')}</tr></thead><tbody>${shown.map((t, i) => `<tr><td><span class="rank">${i + 1}</span></td><td><div class="who"><span class="crest">${esc(initials(t.name))}</span><span><strong>${esc(t.name)}</strong><small>${esc(t.dz)}</small></span></div></td><td class="num">${t.played}</td><td class="num">${t.w}</td>${r.draw ? `<td class="num">${t.d}</td>` : ''}<td class="num">${t.l}</td>${limit ? '' : `<td class="num">${t.gf}</td><td class="num">${t.ga}</td>`}<td class="num">${t.gd > 0 ? '+' : ''}${t.gd}</td><td class="num pts">${t.pts}</td></tr>`).join('')}</tbody></table></div>`;
}

/* ---------------------------------------------------------------- gates */
function loadingBlock(text) {
  return `<section class="panel">${empty(text, 'This only takes a moment.', { ic: 'refresh' })}</section>`;
}
const accountButtons = (light = false) => `${button('Create free account', 'signup', { cls: light ? 'btn-light' : 'btn-primary', ic: 'userplus' })}${button('Sign in', 'signin', { cls: light ? 'btn-ghost' : '', ic: 'login' })}`;
function signedOutPage(label, eyebrow) {
  return `${pageHead(eyebrow, esc(label), '')}<section class="panel">${empty(`Sign in to see ${label.toLowerCase()}`, 'Create a free viewer account in a few seconds, or sign in with the account your league admin gave you.', { ic: 'lock', action: `<div class="hero-actions">${accountButtons()}</div>` })}</section>`;
}
// Returns a full page when the normal page cannot be shown yet, or '' to continue.
// Visitors can read the public pages; their data comes from the BFF's public routes.
function gate(label, eyebrow) {
  if (store.session.status === 'checking') return loadingBlock('Checking your session…');
  if (!signedIn() && !PUBLIC_PAGES.includes(ui.page)) return signedOutPage(label, eyebrow);
  if (store.data.status === 'error' && !loaded()) {
    return `${pageHead(eyebrow, esc(label), '')}<section class="panel">${empty('Could not load saved data', store.data.message || 'Check that the API is running, then try again.', { ic: 'alert', action: button('Try again', 'reload', { cls: 'btn-primary', ic: 'refresh' }) })}</section>`;
  }
  if (!loaded()) return loadingBlock(`Loading ${label.toLowerCase()}…`);
  return '';
}

/* ---------------------------------------------------------------- pages */
function dashboardPage() {
  if (store.session.status === 'checking') return loadingBlock('Checking your session…');
  const blocked = gate('Overview', 'Overview');
  if (blocked) return blocked;
  if (!signedIn()) return publicDashboard();
  if (role() === 'super_admin') return adminDashboard();
  if (role() === 'match_official') return officialDashboard();
  if (role() === 'team_manager') return managerDashboard();
  return viewerDashboard();
}

function hero(img, chip, title, text, actions = '') {
  return `<section class="hero-banner"><img src="./assets/landing/${img}" alt="" decoding="async" /><span class="chip">${icon(ROLE_INFO[role()]?.icon || 'eye')}${esc(chip)}</span><h2>${title}</h2><p>${esc(text)}</p>${actions ? `<div class="hero-actions">${actions}</div>` : ''}</section>`;
}

// The fan dashboard (guests and viewers): pick a sport, see what is live, open a tournament.
// Nothing administrative is shown here.
function fanDashboard(guest) {
  const matches = store.matches.filter(inSportM);
  const live = sortByKickoff(matches.filter((m) => m.status === 'live'));
  const upcoming = nextUp(matches.filter((m) => m.status === 'scheduled')).slice(0, 4);
  const recent = [...matches.filter((m) => m.status === 'completed')].sort((a, b) => (b.kickoff?.valueOf() || 0) - (a.kickoff?.valueOf() || 0)).slice(0, 4);
  const tournaments = store.tournaments.filter(inSportT)
    .sort((a, b) => (b.status === 'in_progress') - (a.status === 'in_progress') || (b.startsAt?.valueOf() || 0) - (a.startsAt?.valueOf() || 0));
  const greeting = guest
    ? `<section class="hero-banner hero-compact"><img src="./assets/landing/night-match.jpg" alt="" decoding="async" /><h2>Follow every <b>match</b>, live</h2><p>Pick a sport, open a tournament, and see its fixtures, live scores and table.</p><div class="hero-actions">${accountButtons(true)}</div></section>`
    : `<section class="hero-banner hero-compact"><img src="./assets/landing/saff-final.jpg" alt="" decoding="async" /><h2>Hi <b>${esc(me().username)}</b></h2><p>Pick a sport, open a tournament, and follow its matches and table.</p></section>`;
  const liveBlock = live.length
    ? `<section class="panel live-panel"><div class="panel-head"><div><h2><span class="live-dot"></span>Live now</h2><p>Scores update automatically</p></div></div><div class="match-list">${live.map((m) => matchCard(m)).join('')}</div></section>`
    : '';
  const list = (rows, text) => (rows.length ? `<div class="match-list compact">${rows.map((m) => matchCard(m, { compact: true })).join('')}</div>` : empty(text, '', { ic: 'calendar' }));
  return `${greeting}
  ${sportFilter()}
  ${liveBlock}
  <div class="section-head"><h2>Tournaments</h2>${tournaments.length > 6 ? '<button type="button" class="link-btn" data-page="tournaments">See all →</button>' : ''}</div>
  ${tournaments.length ? `<div class="card-grid">${tournaments.slice(0, 6).map(tournamentCard).join('')}</div>` : `<section class="panel">${empty('No tournaments in this sport yet', 'Try another sport.', { ic: 'trophy' })}</section>`}
  <section class="grid-2-even">${panel('Latest results', '', list(recent, 'No results yet'), '<button type="button" class="link-btn" data-page="matches" data-tab="completed">All results →</button>')}${panel('Coming up', '', list(upcoming, 'No upcoming matches'), '<button type="button" class="link-btn" data-page="matches" data-tab="scheduled">All fixtures →</button>')}</section>`;
}
const publicDashboard = () => fanDashboard(true);

function adminChecklist() {
  const ready = store.tournaments.filter((t) => approvedIn(t.id).length >= 2);
  const pending = pendingRegistrations().length;
  const items = [
    { done: store.tournaments.length > 0, title: 'Create a tournament', text: 'Set the season, sport, dzongkhag and start date.', action: button('New tournament', 'new-tournament', { cls: 'btn-sm' }) },
    { done: store.venues.length > 0, title: 'Add venues', text: "Fixtures are booked at venues in the tournament's dzongkhag.", action: button('Add venue', 'new-venue', { cls: 'btn-sm' }) },
    { done: ready.length > 0, title: 'Approve at least two teams', text: 'Fixtures are generated only for approved teams in the same tournament.', action: button('Add a team', 'new-team', { cls: 'btn-sm' }) },
    { done: store.registrations.length > 0 && pending === 0, title: 'Review team registrations', text: pending ? `${plural(pending, 'entry', 'entries')} waiting for your decision.` : 'Approve or reject the teams that managers enter.', action: '<button type="button" class="btn btn-sm" data-page="registrations">Review</button>' },
    { done: officials().length > 0, title: 'Create match official accounts', text: 'Officials sign in to score the matches you assign to them.', action: button('Add official', 'new-user', { cls: 'btn-sm', data: { role: 'match_official' } }) },
    { done: store.matches.length > 0, title: 'Generate fixtures', text: 'Round-robin fixtures with venues and kick-off times.', action: button('Generate', 'generate', { cls: 'btn-sm' }) },
    { done: store.matches.some((m) => m.officialId), title: 'Assign officials to matches', text: 'Each match needs one official to enter its score.', action: `<button type="button" class="btn btn-sm" data-page="matches" data-tab="unassigned">Open matches</button>` },
    { done: store.matches.some((m) => m.status === 'completed'), title: 'First result finalised', text: 'Standings update from every final result.', action: '' }
  ];
  const done = items.filter((i) => i.done).length;
  const body = `<div class="progress" aria-label="${done} of ${items.length} steps done"><i style="width:${Math.round((done / items.length) * 100)}%"></i></div>
    <ol class="checklist" style="margin-top:16px">${items.map((item, i) => `<li class="check-item ${item.done ? 'done' : ''}"><span class="check-dot">${item.done ? icon('check') : i + 1}</span><span><strong>${esc(item.title)}</strong><small>${esc(item.text)}</small></span>${item.done ? '<span class="badge badge-team_manager">Done</span>' : item.action}</li>`).join('')}</ol>`;
  return panel('Season set-up', `${done} of ${items.length} steps done`, body);
}

function adminDashboard() {
  const openT = store.tournaments.filter((t) => t.status !== 'completed').length;
  const completed = store.matches.filter((m) => m.status === 'completed').length;
  const dz = new Set(store.teams.map((t) => t.dzongkhag).filter(Boolean)).size;
  const upcoming = nextUp(store.matches).slice(0, 5);
  const unassigned = sortByKickoff(store.matches.filter((m) => !m.officialId && m.status !== 'completed'));
  const officialsCount = store.usersStatus === 'loaded' ? String(officials().length) : '—';
  const pending = pendingRegistrations().length;
  return `${hero('stadium-dzong.jpg', 'Super Admin', `Welcome back, <b>${esc(me().username)}</b>`, 'Set up tournaments, accounts and fixtures, then follow every result as officials finalise it.', `${button('New tournament', 'new-tournament', { cls: 'btn-light', ic: 'plus' })}${button('Schedule a match', 'new-match', { cls: 'btn-ghost', ic: 'calendar' })}`)}
  ${pending ? notice(`<b>${plural(pending, 'team entry', 'team entries')}</b> waiting for your approval. <button type="button" class="link-btn" data-page="registrations">Review registrations →</button>`, true) : ''}
  <section class="metrics">${metric('Tournaments', store.tournaments.length, `${openT} open or in progress`, 'trophy')}${metric('Teams', store.teams.length, `from ${plural(dz, 'dzongkhag')}`, 'users')}${metric('Matches', `${completed}/${store.matches.length}`, 'finalised / scheduled', 'calendar')}${metric('Match officials', officialsCount, 'accounts that can score', 'whistle')}</section>
  <section class="grid-2">${adminChecklist()}${panel('Coming up', upcoming.length ? 'Next scheduled and live matches' : 'Nothing scheduled yet', upcoming.length ? `<div class="match-list compact">${upcoming.map((m) => matchCard(m, { compact: true })).join('')}</div>` : empty('No upcoming matches', 'Generate fixtures for a tournament with at least two teams.', { ic: 'calendar' }), '<button type="button" class="link-btn" data-page="matches">All matches →</button>')}</section>
  <section class="grid-2-even">${standingsPreview()}${panel('Needs an official', unassigned.length ? `${plural(unassigned.length, 'match', 'matches')} without an official` : 'Every match has an official', unassigned.length ? `<div class="match-list compact">${unassigned.slice(0, 4).map((m) => matchCard(m, { compact: true })).join('')}</div>` : empty('All set', 'Every scheduled match has an official assigned.', { ic: 'check' }), unassigned.length ? '<button type="button" class="link-btn" data-page="matches" data-tab="unassigned">Assign officials →</button>' : '')}</section>`;
}

function officialDashboard() {
  const mine = sortByKickoff(store.matches.filter(isMine));
  const live = mine.filter((m) => m.status === 'live').length;
  const toPlay = mine.filter((m) => m.status === 'scheduled').length;
  const done = mine.filter((m) => m.status === 'completed').length;
  const open = mine.filter((m) => m.status !== 'completed');
  return `${hero('goalkeeper-dive.jpg', 'Match Official', `Ready for kick-off, <b>${esc(me().username)}</b>`, 'These are the matches assigned to you. Save the score while play is on, then finalise the result at full time.')}
  <section class="metrics">${metric('Assigned to you', mine.length, 'matches in total', 'whistle')}${metric('Live now', live, 'scores you are updating', 'chart')}${metric('To play', toPlay, 'not started yet', 'calendar')}${metric('Finalised', done, 'results locked', 'check')}</section>
  <section class="grid-2">${panel('Your matches', open.length ? 'Open a match to enter its score' : 'Nothing waiting for a score', open.length ? `<div class="match-list">${open.map((m) => matchCard(m)).join('')}</div>` : empty(mine.length ? 'All your matches are finalised' : 'No matches assigned yet', mine.length ? 'Finalised results appear in the standings.' : 'The league admin assigns officials to matches. Once you are assigned, the match appears here.', { ic: 'whistle' }))}
  ${panel('How scoring works', '', `<ol class="steps" style="grid-template-columns:1fr"><li><strong>Start scoring</strong>Open your match when play begins.</li><li><strong>Save the live score</strong>Update it as goals go in; the result stays open.</li><li><strong>Finalise at full time</strong>The result locks and the standings update. Only an admin correction can change it later.</li></ol>`)}</section>`;
}

function managerDashboard() {
  const teams = myTeams();
  const ids = new Set(teams.map((t) => t.id));
  const fixtures = nextUp(store.matches.filter((m) => ids.has(m.homeId) || ids.has(m.awayId))).slice(0, 5);
  const stats = teams.map((t) => teamStats(t.id));
  const played = stats.reduce((n, s) => n + s.p, 0);
  const points = stats.reduce((n, s) => n + s.pts, 0);
  const entries = store.registrations.filter((r) => ids.has(r.teamId));
  const approved = entries.filter((r) => r.status === 'approved').length;
  const waiting = entries.filter((r) => r.status === 'pending').length;
  const rejected = entries.filter((r) => r.status === 'rejected').length;
  const table = teams.length ? `<div class="table-wrap"><table class="table"><thead><tr><th>Team</th><th>Entry</th><th></th></tr></thead><tbody>${teams.map((t) => `<tr><td><div class="who"><span class="crest">${esc(t.short)}</span><span><strong>${esc(t.name)}</strong><small>${esc(tournamentName(t.tournamentId))}</small></span></div></td><td>${entryCell(t)}</td><td class="row-actions">${teamActions(t, { compact: true })}</td></tr>`).join('')}</tbody></table></div>`
    : empty('You have not entered a team yet', 'Create your team and enter it into a tournament that is open for registration. The league admin then approves the entry.', { ic: 'users', action: button('Create a team', 'new-team', { cls: 'btn-primary', ic: 'plus' }) });
  const steps = '<ol class="steps" style="grid-template-columns:1fr"><li><strong>Create your team</strong>Pick the tournament you want to enter; the team is linked to your account.</li><li><strong>Wait for approval</strong>The league admin approves the entry. Only approved teams get fixtures.</li><li><strong>Build your squad</strong>Add your players and shirt numbers on the Squads page.</li><li><strong>Play</strong>Your fixtures, results and points appear here.</li></ol>';
  return `${hero('bhutan-maldives.jpg', 'Team Manager', `Your teams, <b>${esc(me().username)}</b>`, 'Enter your teams into open tournaments, build your squads, then follow fixtures, results and points.', button('Create a team', 'new-team', { cls: 'btn-light', ic: 'plus' }))}
  ${waiting ? notice(`<b>${plural(waiting, 'entry', 'entries')}</b> waiting for the league admin to approve. Fixtures are generated only for approved teams.`) : ''}
  ${rejected ? notice(`<b>${plural(rejected, 'entry was', 'entries were')} rejected.</b> Ask the league admin for the reason. <button type="button" class="link-btn" data-page="registrations">See entries →</button>`, true) : ''}
  <section class="metrics">${metric('Your teams', teams.length, 'linked to your account', 'users')}${metric('Approved', approved, waiting ? `${waiting} waiting` : 'tournament entries', 'flag')}${metric('Upcoming', fixtures.length, 'matches to play', 'calendar')}${metric('Points', points, `from ${plural(played, 'match', 'matches')}`, 'chart')}</section>
  <section class="grid-2">${panel('My teams', teams.length ? 'Entry status and squad' : '', table)}${panel('Next fixtures', '', fixtures.length ? `<div class="match-list compact">${fixtures.map((m) => matchCard(m, { compact: true })).join('')}</div>` : empty('No fixtures yet', 'Fixtures appear after your entry is approved and the admin generates them.', { ic: 'calendar' }))}</section>
  ${teams.length ? '' : panel('How it works for team managers', '', steps)}`;
}

// Entry status for a team in its own tournament, for roles that can see registrations.
function entryCell(team) {
  if (!tracksRegistrations()) return '';
  const reg = registrationFor(team.id, team.tournamentId);
  return registrationBadge(reg);
}
// Row buttons: open the squad; enter the team when it has no entry and the role may register it.
function teamActions(team, { compact = false } = {}) {
  const squad = !signedIn() ? ''
    : compact ? `<button type="button" class="btn btn-sm btn-icon" data-page="squads" data-team="${esc(team.id)}" title="Squad" aria-label="${esc(team.name)} squad">${icon('shirt')}</button>`
      : `<button type="button" class="btn btn-sm" data-page="squads" data-team="${esc(team.id)}">${icon('shirt')}Squad</button>`;
  const tournament = byId(store.tournaments, team.tournamentId);
  const reg = registrationFor(team.id, team.tournamentId);
  // A team is entered once per tournament; the registration service refuses a second entry.
  const mayEnter = tracksRegistrations() && tournament && tournament.status === 'registration_open' && !reg
    && (role() === 'super_admin' || team.managerId === me().id);
  const label = compact ? 'Enter' : role() === 'super_admin' ? 'Enter & approve' : 'Enter tournament';
  const enter = mayEnter ? button(label, 'register-team', { cls: 'btn-sm btn-primary', ic: 'flag', data: { team: team.id, tournament: team.tournamentId }, roleAction: 'register-team' }) : '';
  return `${enter}${squad}`;
}

const viewerDashboard = () => fanDashboard(false);

// One tournament: live and upcoming matches, results and the table, in one place.
function tournamentPage() {
  const blocked = gate('Tournament', 'Tournament');
  if (blocked) return blocked;
  const t = byId(store.tournaments, ui.tournamentId);
  if (!t) return `${pageHead('Tournaments', 'Tournament not found', '')}<section class="panel">${empty('This tournament is not available', 'Pick one from the list.', { ic: 'trophy', action: '<button type="button" class="btn btn-primary" data-page="tournaments">All tournaments</button>' })}</section>`;
  if (store.standings.tournamentId !== t.id) setTimeout(() => window.DrukAPI?.loadStandings(t.id), 0);
  const matches = store.matches.filter((m) => m.tournamentId === t.id);
  const live = sortByKickoff(matches.filter((m) => m.status === 'live'));
  const upcoming = sortByKickoff(matches.filter((m) => m.status === 'scheduled'));
  const results = [...matches.filter((m) => m.status === 'completed')].sort((a, b) => (b.kickoff?.valueOf() || 0) - (a.kickoff?.valueOf() || 0));
  const s = store.standings;
  const rows = s.tournamentId === t.id ? s.rows : [];
  const tabs = [['matches', 'Matches', live.length + upcoming.length], ['results', 'Results', results.length], ['table', 'Table', rows.length]];
  if (!tabs.some(([k]) => k === ui.tTab)) ui.tTab = 'matches';
  const list = (ms, text) => (ms.length ? `<div class="match-list">${ms.map((m) => matchCard(m)).join('')}</div>` : `<section class="panel">${empty(text, '', { ic: 'calendar' })}</section>`);
  let body;
  if (ui.tTab === 'results') body = list(results, 'No results yet — they appear when officials finalise matches.');
  else if (ui.tTab === 'table') {
    body = `<section class="panel">${s.tournamentId !== t.id || s.status === 'loading' ? empty('Loading the table…', '', { ic: 'refresh' }) : s.status === 'error' ? empty('Could not load the table', s.message, { ic: 'alert' }) : rows.length ? standingsTable(rows, 0, t.sport) : empty('No table yet', `The table builds from finalised results: ${rulesFor(t.sport).points}.`, { ic: 'chart' })}</section>
    ${rows.length ? `<p class="standings-foot">${esc(standingsLegend(t.sport))}</p>` : ''}`;
  } else {
    body = `${live.length ? `<section class="panel live-panel"><div class="panel-head"><div><h2><span class="live-dot"></span>Live now</h2><p>Scores update automatically</p></div></div><div class="match-list">${live.map((m) => matchCard(m)).join('')}</div></section>` : ''}
    ${upcoming.length || !live.length ? `<div class="section-head"><h2>Upcoming</h2></div>${list(upcoming, 'No upcoming matches.')}` : ''}`;
  }
  const download = rows.length ? `<a class="btn btn-light" href="./fn/standings-csv?tournament_id=${encodeURIComponent(t.id)}" download>${icon('download')}Table as CSV</a>` : '';
  return `<button type="button" class="link-btn back-link" data-page="${signedIn() && role() !== 'viewer' ? 'tournaments' : 'dashboard'}">← Back</button>
  <section class="hero-banner t-hero"><img src="./assets/landing/${coverFor(t)}.jpg" alt="" decoding="async" />${sportChip(t.sport)}<h2>${esc(t.name)}</h2><p>${icon('pin')} ${esc(t.dzongkhag || '')} · ${esc(formatDate(t.startsAt))} · ${plural(teamsIn(t.id).length, 'team')}</p><div class="hero-actions">${tournamentStatus(t)}${download}</div></section>
  <div class="toolbar"><div class="tabs">${tabs.map(([k, l, n]) => `<button type="button" class="tab ${ui.tTab === k ? 'active' : ''}" data-action="t-tab" data-tab="${k}">${l}<span class="count">${n}</span></button>`).join('')}</div></div>
  ${body}`;
}

function standingsPreview() {
  const s = store.standings;
  const name = tournamentName(s.tournamentId);
  let body;
  if (!store.tournaments.length) body = empty('No tournaments yet', '', { ic: 'trophy' });
  else if (s.status === 'loading') body = empty('Loading standings…', '', { ic: 'refresh' });
  else if (s.status === 'error') body = empty('Could not load standings', s.message, { ic: 'alert' });
  else if (!s.rows.length) body = empty('No standings yet', 'Standings appear after the first result is finalised.', { ic: 'chart' });
  else body = standingsTable(s.rows, 5, byId(store.tournaments, s.tournamentId)?.sport);
  return panel('Standings', store.tournaments.length ? name : '', body, '<button type="button" class="link-btn" data-page="standings">Full table →</button>');
}

function tournamentsPage() {
  const blocked = gate('Tournaments', 'Competitions');
  if (blocked) return blocked;
  const seasons = [...new Set(store.tournaments.map((t) => String(t.season || '')).filter(Boolean))]
    .sort((a, b) => a.localeCompare(b, undefined, { numeric: true })).reverse();
  const q = ui.query.toLowerCase();
  const list = store.tournaments.filter((t) => inSportT(t) && (ui.season === 'all' || String(t.season) === ui.season) && `${t.name} ${t.sport} ${t.dzongkhag}`.toLowerCase().includes(q));
  // Grouped by sport so football, futsal, archery and the rest are never mixed together.
  const groups = sportsPresent().concat(list.some((t) => !sportKey(t.sport)) ? [''] : [])
    .map((k) => [k, list.filter((t) => sportKey(t.sport) === k)]).filter(([, ts]) => ts.length);
  const cards = groups.map(([k, ts]) => `<div class="sport-heading"><span aria-hidden="true">${sportInfo(k).emoji}</span><h2>${esc(sportInfo(k).label)}</h2><span class="count">${plural(ts.length, 'tournament')}</span></div><div class="card-grid">${ts.map(tournamentCard).join('')}</div>`).join('');
  const emptyState = store.tournaments.length
    ? empty('No tournaments match', 'Change the season or the search.', { ic: 'search' })
    : empty('No tournaments yet', canRoleAction('new-tournament') ? 'Create the first tournament of the season.' : 'Tournaments appear here once the league admin creates them.', { ic: 'trophy', action: button('New tournament', 'new-tournament', { cls: 'btn-primary', ic: 'plus', roleAction: 'new-tournament' }) });
  return `${pageHead('Competitions', 'Tournaments', 'Pick a sport, then open a tournament to see its matches and table.', button('New tournament', 'new-tournament', { cls: 'btn-primary', ic: 'plus', roleAction: 'new-tournament' }))}
  ${sportFilter()}
  <div class="toolbar"><label class="select-wrap">Season<select id="season-filter"><option value="all">All seasons</option>${seasons.map((s) => `<option value="${esc(s)}" ${ui.season === s ? 'selected' : ''}>${esc(s)}</option>`).join('')}</select></label><span class="spacer"></span><label class="search">${icon('search')}<input id="page-search" placeholder="Search tournaments…" value="${esc(ui.query)}" /></label></div>
  ${list.length ? cards : `<section class="panel">${emptyState}</section>`}`;
}

function teamsPage() {
  const blocked = gate('Teams', 'Teams & dzongkhags');
  if (blocked) return blocked;
  const isManager = role() === 'team_manager';
  const scope = isManager ? ui.teamScope : 'all';
  const base = scope === 'mine' ? myTeams() : store.teams;
  const q = ui.query.toLowerCase();
  const rows = base.filter((t) => `${t.name} ${t.dzongkhag} ${tournamentName(t.tournamentId)}`.toLowerCase().includes(q));
  const showManager = role() === 'super_admin';
  const dz = new Set(store.teams.map((t) => t.dzongkhag).filter(Boolean)).size;
  const showEntry = tracksRegistrations();
  const showActions = signedIn();
  const table = rows.length ? `<section class="panel"><div class="table-wrap"><table class="table"><thead><tr><th>Team</th>${showManager ? '<th>Manager</th>' : ''}${showEntry ? '<th>Entry</th>' : ''}<th class="num">P</th><th class="num">W</th><th class="num">D</th><th class="num">L</th><th class="num">GD</th><th class="num">Pts</th>${showActions ? '<th></th>' : ''}</tr></thead><tbody>${rows.map((t) => {
    const s = teamStats(t.id);
    const mine = isManager && t.managerId === me().id;
    // A manager sees entry status only for their own teams (registrations are private to them).
    const entry = showEntry ? `<td>${!isManager || mine ? entryCell(t) : ''}</td>` : '';
    return `<tr><td><div class="who"><span class="crest">${esc(t.short)}</span><span><strong>${esc(t.name)}</strong>${mine ? ' <span class="badge badge-team_manager">Yours</span>' : ''}<small>${esc(t.dzongkhag)} · ${esc(tournamentName(t.tournamentId))}</small></span></div></td>${showManager ? `<td>${t.managerId ? esc(userName(t.managerId) || 'Team manager') : '<span class="chip">Admin-created</span>'}</td>` : ''}${entry}<td class="num">${s.p}</td><td class="num">${s.w}</td><td class="num">${s.d}</td><td class="num">${s.l}</td><td class="num">${s.gd > 0 ? '+' : ''}${s.gd}</td><td class="num pts">${s.pts}</td>${showActions ? `<td class="row-actions">${teamActions(t, { compact: true })}</td>` : ''}</tr>`;
  }).join('')}</tbody></table></div></section>`
    : `<section class="panel">${scope === 'mine' && !myTeams().length
      ? empty('You have not entered a team yet', 'Create your team and enter it into a tournament that is open for registration.', { ic: 'users', action: button('Create a team', 'new-team', { cls: 'btn-primary', ic: 'plus' }) })
      : empty(store.teams.length ? 'No teams match the search' : 'No teams yet', store.teams.length ? '' : 'Teams appear here once they are registered for a tournament.', { ic: 'users', action: button('Add a team', 'new-team', { cls: 'btn-primary', ic: 'plus', roleAction: 'new-team' }) })}</section>`;
  const tabs = isManager ? `<div class="tabs">${[['mine', 'My teams', myTeams().length], ['all', 'All teams', store.teams.length]].map(([k, l, n]) => `<button type="button" class="tab ${ui.teamScope === k ? 'active' : ''}" data-action="team-scope" data-scope="${k}">${l}<span class="count">${n}</span></button>`).join('')}</div>` : '';
  return `${pageHead('Teams & dzongkhags', 'Teams', 'Teams in each tournament, with their record from finalised results.', button(isManager ? 'Create a team' : 'Add a team', 'new-team', { cls: 'btn-primary', ic: 'plus', roleAction: 'new-team' }))}
  ${signedIn() ? '' : notice('You are browsing as a guest. <button type="button" class="link-btn" data-action="signup">Create a free account</button> to see each team\'s squad.')}
  <section class="metrics">${metric('Teams', store.teams.length, 'registered', 'users')}${metric('Dzongkhags', dz, 'represented', 'pin')}${metric('Tournaments', new Set(store.teams.map((t) => t.tournamentId).filter(Boolean)).size, 'with teams', 'trophy')}${metric('Results', store.matches.filter((m) => m.status === 'completed').length, 'finalised matches', 'check')}</section>
  <div class="toolbar">${tabs}<span class="spacer"></span><label class="search">${icon('search')}<input id="page-search" placeholder="Search teams, dzongkhags…" value="${esc(ui.query)}" /></label></div>${table}`;
}

function matchesPage() {
  const blocked = gate('Matches', 'On the pitch');
  if (blocked) return blocked;
  const r = role();
  const q = ui.query.toLowerCase();
  const sportTournaments = store.tournaments.filter(inSportT);
  if (ui.matchTournament !== 'all' && !sportTournaments.some((t) => t.id === ui.matchTournament)) ui.matchTournament = 'all';
  const inTournament = store.matches.filter((m) => inSportM(m) && (ui.matchTournament === 'all' || m.tournamentId === ui.matchTournament));
  const tabs = [['all', 'All'], ['live', 'Live'], ['scheduled', 'Upcoming'], ['completed', 'Results']];
  if (r === 'match_official') tabs.unshift(['mine', 'Assigned to me']);
  if (canRoleAction('assign-official')) tabs.push(['unassigned', 'Needs official']);
  const filters = {
    all: () => true, live: (m) => m.status === 'live', scheduled: (m) => m.status === 'scheduled', completed: (m) => m.status === 'completed',
    mine: isMine, unassigned: (m) => !m.officialId && m.status !== 'completed'
  };
  if (!filters[ui.matchTab] || !tabs.some(([k]) => k === ui.matchTab)) ui.matchTab = 'all';
  const list = sortByKickoff(inTournament.filter((match) => filters[ui.matchTab](match)).filter((m) => `${m.home} ${m.away} ${m.venue}`.toLowerCase().includes(q)));
  const listHtml = list.length ? `<div class="match-list">${list.map((m) => matchCard(m)).join('')}</div>`
    : `<section class="panel">${store.matches.length
      ? empty(ui.matchTab === 'mine' ? 'No matches assigned to you' : 'No matches here', ui.matchTab === 'mine' ? 'The league admin assigns officials to matches. Assigned matches appear here.' : 'Try another tab, tournament or search.', { ic: 'calendar' })
      : empty('No matches yet', canRoleAction('generate-fixtures') ? 'Generate fixtures for a tournament with two or more teams, or schedule a single match.' : 'Fixtures appear once the league admin generates them.', { ic: 'calendar', action: button('Generate fixtures', 'generate', { cls: 'btn-primary', roleAction: 'generate-fixtures' }) })}</section>`;
  const intro = r === 'match_official' ? notice('You can enter scores only for matches <b>assigned to you</b>. Saving keeps the result open; finalising locks it and updates the standings.')
    : r === 'super_admin' ? notice('Assign an official to every match. Only the assigned official can enter and finalise its score.') : '';
  return `${pageHead('On the pitch', 'Matches &amp; scores', 'Fixtures, live scores and final results.', `${button('Schedule a match', 'new-match', { ic: 'calendar', roleAction: 'schedule-match' })}${button('Generate fixtures', 'generate', { cls: 'btn-primary', ic: 'plus', roleAction: 'generate-fixtures' })}`)}
  ${intro}
  ${sportFilter()}
  <div class="toolbar"><div class="tabs">${tabs.map(([k, l]) => `<button type="button" class="tab ${ui.matchTab === k ? 'active' : ''}" data-action="match-tab" data-tab="${k}">${l}<span class="count">${inTournament.filter((match) => filters[k](match)).length}</span></button>`).join('')}</div><span class="spacer"></span>
  <label class="select-wrap">Tournament<select id="match-tournament"><option value="all">All tournaments</option>${sportTournaments.map((t) => `<option value="${esc(t.id)}" ${ui.matchTournament === t.id ? 'selected' : ''}>${esc(t.name)}</option>`).join('')}</select></label>
  <label class="search">${icon('search')}<input id="page-search" placeholder="Search teams or venues…" value="${esc(ui.query)}" /></label></div>${listHtml}`;
}

function standingsPage() {
  const blocked = gate('Standings', 'The race for the title');
  if (blocked) return blocked;
  const s = store.standings;
  let body;
  if (!store.tournaments.length) body = empty('No tournaments yet', 'Standings appear once a tournament has finalised results.', { ic: 'trophy' });
  else if (s.status === 'loading') body = empty('Loading standings…', '', { ic: 'refresh' });
  else if (s.status === 'error') body = empty('Could not load standings', s.message, { ic: 'alert', action: button('Try again', 'reload-standings', { cls: 'btn-primary' }) });
  else if (!s.rows.length) body = empty('No standings yet', 'Finalised match results build this table: three points for a win, one for a draw.', { ic: 'chart' });
  else body = standingsTable(s.rows, 0, byId(store.tournaments, s.tournamentId)?.sport);
  const updated = s.status === 'loaded' && s.updatedAt ? `Updated ${new Date(s.updatedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}` : '';
  // The CSV comes from the standings-csv serverless function (public, read-only).
  const download = s.tournamentId ? `<a class="btn" href="./fn/standings-csv?tournament_id=${encodeURIComponent(s.tournamentId)}" download>${icon('download')}Download CSV</a>` : '';
  const choices = store.tournaments.filter((t) => inSportT(t) || t.id === s.tournamentId);
  return `${pageHead('The race for the title', 'Standings', 'Built only from finalised results.', download)}
  ${sportFilter()}
  <div class="toolbar"><label class="select-wrap">Tournament<select id="standings-tournament">${choices.map((t) => `<option value="${esc(t.id)}" ${s.tournamentId === t.id ? 'selected' : ''}>${esc(t.name)} · ${esc(t.season || '')}</option>`).join('')}</select></label><span class="spacer"></span>${updated ? `<span class="chip">${icon('refresh')}${esc(updated)}</span>` : ''}</div>
  <section class="panel"><div class="panel-head"><div><h2>${esc(tournamentName(s.tournamentId))}</h2><p>${s.rows.length ? plural(s.rows.length, 'team') : 'League table'}</p></div></div>${body}</section>
  <p class="standings-foot">${esc(standingsLegend(byId(store.tournaments, s.tournamentId)?.sport))}</p>`;
}

function usersNotice() {
  if (store.usersStatus === 'error') return notice('Could not load the account list. Check that the User service is running, then reload.', true);
  if (store.usersStatus !== 'loaded') return notice('Loading accounts…');
  return '';
}

function officialsPage() {
  const blocked = gate('Officials', 'The people behind the game');
  if (blocked) return blocked;
  const list = officials().filter((u) => u.username.toLowerCase().includes(ui.query.toLowerCase()));
  const unassigned = store.matches.filter((m) => !m.officialId && m.status !== 'completed').length;
  const cards = list.map((u) => {
    const assigned = store.matches.filter((m) => m.officialId === u.id);
    return `<article class="person"><div class="person-top"><span class="avatar">${esc(initials(u.username))}</span><span><strong>${esc(u.username)}</strong>${roleBadge(u.role)}</span></div>
      <div class="person-stats"><span><b>${assigned.length}</b>assigned</span><span><b>${assigned.filter((m) => m.status !== 'completed').length}</b>to play</span><span><b>${assigned.filter((m) => m.status === 'completed').length}</b>finalised</span></div></article>`;
  }).join('');
  return `${pageHead('The people behind the game', 'Officials', 'Match official accounts and the matches assigned to them.', button('Add official', 'new-user', { cls: 'btn-primary', ic: 'plus', data: { role: 'match_official' }, roleAction: 'manage-users' }))}
  ${usersNotice()}${unassigned ? notice(`${plural(unassigned, 'match', 'matches')} still need an official. <button type="button" class="link-btn" data-page="matches" data-tab="unassigned">Assign officials →</button>`, true) : ''}
  <div class="toolbar"><span class="spacer"></span><label class="search">${icon('search')}<input id="page-search" placeholder="Search officials…" value="${esc(ui.query)}" /></label></div>
  ${list.length ? `<div class="people-grid">${cards}</div>` : `<section class="panel">${empty(officials().length ? 'No officials match the search' : 'No official accounts yet', 'Create a match official account, then assign the official to matches from the Matches page.', { ic: 'whistle', action: button('Add official', 'new-user', { cls: 'btn-primary', data: { role: 'match_official' }, roleAction: 'manage-users' }) })}</section>`}`;
}

function usersPage() {
  const blocked = gate('Users & roles', 'Accounts');
  if (blocked) return blocked;
  const roles = ['all', 'super_admin', 'match_official', 'team_manager', 'viewer'];
  const list = store.users.filter((u) => (ui.userRole === 'all' || u.role === ui.userRole) && u.username.toLowerCase().includes(ui.query.toLowerCase()));
  const rows = list.map((u) => `<tr><td><div class="who"><span class="avatar">${esc(initials(u.username))}</span><span><strong>${esc(u.username)}</strong>${u.id === me().id ? ' <span class="chip">You</span>' : ''}</span></div></td><td>${roleBadge(u.role)}</td><td>${esc((ROLE_INFO[u.role]?.can || []).join(' · '))}</td></tr>`).join('');
  return `${pageHead('Accounts', 'Users &amp; roles', 'Create accounts for officials, team managers and viewers. The role decides what each person can do.', button('Create account', 'new-user', { cls: 'btn-primary', ic: 'plus', roleAction: 'manage-users' }))}
  ${usersNotice()}${notice('Accounts cannot be deleted or have their password reset from the web app yet. Give each person their username and password directly.')}
  <div class="toolbar"><div class="tabs">${roles.map((r) => `<button type="button" class="tab ${ui.userRole === r ? 'active' : ''}" data-action="user-role" data-role="${r}">${r === 'all' ? 'All' : esc(roleLabels[r])}<span class="count">${r === 'all' ? store.users.length : store.users.filter((u) => u.role === r).length}</span></button>`).join('')}</div><span class="spacer"></span><label class="search">${icon('search')}<input id="page-search" placeholder="Search usernames…" value="${esc(ui.query)}" /></label></div>
  <section class="panel">${list.length ? `<div class="table-wrap"><table class="table"><thead><tr><th>Account</th><th>Role</th><th>Can</th></tr></thead><tbody>${rows}</tbody></table></div>` : empty('No accounts here', 'Create an account to get started.', { ic: 'users' })}</section>`;
}

function venuesPage() {
  const blocked = gate('Venues', 'Where it happens');
  if (blocked) return blocked;
  const usage = (name) => {
    const games = store.matches.filter((m) => (m.venue || '').trim().toLowerCase() === name.toLowerCase());
    const next = sortByKickoff(games.filter((m) => m.status !== 'completed' && m.kickoff))[0];
    return { total: games.length, done: games.filter((m) => m.status === 'completed').length, next: next?.kickoff || null };
  };
  const q = ui.query.toLowerCase();
  const list = [...store.venues].sort((a, b) => a.dzongkhag.localeCompare(b.dzongkhag) || a.name.localeCompare(b.name)).filter((v) => `${v.name} ${v.dzongkhag}`.toLowerCase().includes(q));
  const rows = list.map((v) => {
    const u = usage(v.name);
    return `<tr><td><div class="who"><span class="crest">${icon('pin')}</span><span><strong>${esc(v.name)}</strong><small>${esc(v.dzongkhag)}</small></span></div></td><td class="num">${v.capacity ? v.capacity.toLocaleString() : '—'}</td><td class="num">${u.total}</td><td class="num">${u.done}</td><td>${u.next ? esc(`${dateParts(u.next).long}, ${dateParts(u.next).time}`) : '—'}</td></tr>`;
  }).join('');
  const covered = new Set(store.venues.map((v) => v.dzongkhag));
  const missing = store.tournaments.filter((t) => t.status !== 'completed' && t.dzongkhag && !covered.has(t.dzongkhag));
  const status = store.venuesStatus === 'error' ? notice('Could not load venues. Check that the Venue service is running, then reload.', true) : store.venuesStatus !== 'loaded' ? notice('Loading venues…') : '';
  return `${pageHead('Where it happens', 'Venues', 'Grounds that fixtures can be booked at. Generated fixtures use the venues in the tournament\'s dzongkhag.', button('Add venue', 'new-venue', { cls: 'btn-primary', ic: 'plus', roleAction: 'manage-venues' }))}
  ${status}${missing.length ? notice(`No venue yet in ${missing.map((t) => `<b>${esc(t.dzongkhag)}</b>`).filter((v, i, a) => a.indexOf(v) === i).join(', ')} — add one before generating fixtures for ${esc(missing.map((t) => t.name).join(', '))}.`, true) : ''}
  <section class="metrics">${metric('Venues', store.venues.length, `in ${plural(covered.size, 'dzongkhag')}`, 'pin')}${metric('Capacity', store.venues.reduce((n, v) => n + v.capacity, 0).toLocaleString(), 'seats in total', 'users')}${metric('Matches', store.matches.filter((m) => m.venue).length, 'with a venue', 'calendar')}${metric('Played', store.matches.filter((m) => m.venue && m.status === 'completed').length, 'finalised there', 'check')}</section>
  <div class="toolbar"><span class="spacer"></span><label class="search">${icon('search')}<input id="page-search" placeholder="Search venues, dzongkhags…" value="${esc(ui.query)}" /></label></div>
  <section class="panel">${rows ? `<div class="table-wrap"><table class="table"><thead><tr><th>Venue</th><th class="num">Capacity</th><th class="num">Matches</th><th class="num">Played</th><th>Next match</th></tr></thead><tbody>${rows}</tbody></table></div>` : empty(store.venues.length ? 'No venues match the search' : 'No venues yet', 'Add the grounds in each dzongkhag that hosts a tournament.', { ic: 'pin', action: button('Add venue', 'new-venue', { cls: 'btn-primary', ic: 'plus', roleAction: 'manage-venues' }) })}</section>`;
}

// Squads: everyone signed in can read a team's players; the admin and that team's manager can edit.
function squadsPage() {
  const blocked = gate('Squads', 'Players');
  if (blocked) return blocked;
  const isManager = role() === 'team_manager';
  const choices = isManager && myTeams().length ? [...myTeams(), ...store.teams.filter((t) => !myTeams().includes(t))] : store.teams;
  if (!choices.length) {
    return `${pageHead('Players', 'Squads', 'Each team\'s players and shirt numbers.', '')}<section class="panel">${empty('No teams yet', isManager ? 'Create your team first, then add its players here.' : 'Squads appear once teams are entered.', { ic: 'shirt', action: isManager ? button('Create a team', 'new-team', { cls: 'btn-primary', ic: 'plus' }) : '' })}</section>`;
  }
  if (!byId(store.teams, ui.squadTeam)) ui.squadTeam = choices[0].id;
  const team = byId(store.teams, ui.squadTeam);
  const squad = store.players[team.id];
  if (!squad) setTimeout(() => window.DrukAPI?.loadPlayers(team.id), 0);
  const editable = canEditSquad(team);
  const q = ui.query.toLowerCase();
  const players = (squad?.rows || []).filter((p) => `${p.given} ${p.family} ${p.shirt}`.toLowerCase().includes(q))
    .sort((a, b) => (Number(a.shirt) || 999) - (Number(b.shirt) || 999) || a.given.localeCompare(b.given));
  let body;
  if (!squad || (squad.status === 'loading' && !squad.rows.length)) body = empty('Loading the squad…', '', { ic: 'refresh' });
  else if (squad.status === 'error') body = empty('Could not load the squad', squad.message, { ic: 'alert', action: button('Try again', 'reload-squad', { cls: 'btn-primary', data: { team: team.id } }) });
  else if (!squad.rows.length) body = empty('No players yet', editable ? 'Add each player with their shirt number.' : 'The team manager has not added players yet.', { ic: 'shirt', action: editable ? button('Add player', 'new-player', { cls: 'btn-primary', ic: 'plus', data: { team: team.id } }) : '' });
  else if (!players.length) body = empty('No players match the search', '', { ic: 'search' });
  else body = `<div class="table-wrap"><table class="table"><thead><tr><th class="shirt-col">No.</th><th>Player</th>${editable ? '<th></th>' : ''}</tr></thead><tbody>${players.map((p) => `<tr><td class="shirt-col"><span class="shirt">${esc(p.shirt || '–')}</span></td><td><div class="who"><span class="avatar">${esc(initials(`${p.given} ${p.family}`))}</span><strong>${esc(`${p.given} ${p.family}`.trim())}</strong></div></td>${editable ? `<td class="row-actions"><button type="button" class="btn btn-sm btn-icon" data-action="edit-player" data-team="${esc(team.id)}" data-player="${esc(p.id)}" title="Edit player" aria-label="Edit ${esc(`${p.given} ${p.family}`.trim())}">${icon('edit')}</button></td>` : ''}</tr>`).join('')}</tbody></table></div>`;
  const options = choices.map((t) => `<option value="${esc(t.id)}" ${t.id === team.id ? 'selected' : ''}>${esc(t.name)}${isManager && t.managerId === me().id ? ' (yours)' : ''}</option>`).join('');
  const reg = tracksRegistrations() && (!isManager || team.managerId === me().id) ? registrationBadge(registrationFor(team.id, team.tournamentId)) : '';
  return `${pageHead('Players', 'Squads', editable ? 'Add your players and their shirt numbers. Shirt numbers run from 1 to 99.' : 'Each team\'s players and shirt numbers.', editable ? button('Add player', 'new-player', { cls: 'btn-primary', ic: 'plus', data: { team: team.id } }) : '')}
  <div class="toolbar"><label class="select-wrap">Team<select id="squad-team">${options}</select></label><span class="spacer"></span><label class="search">${icon('search')}<input id="page-search" placeholder="Search players…" value="${esc(ui.query)}" /></label></div>
  <section class="panel"><div class="panel-head"><div class="who"><span class="crest">${esc(team.short)}</span><span><h2>${esc(team.name)}</h2><p>${esc(team.dzongkhag)} · ${esc(tournamentName(team.tournamentId))} · ${plural(squad?.rows.length || 0, 'player')}</p></span></div>${reg}</div>${body}</section>`;
}

// Registrations: the admin approves or rejects entries; a team manager follows their own.
function registrationsPage() {
  const blocked = gate('Registrations', 'Tournament entries');
  if (blocked) return blocked;
  const admin = role() === 'super_admin';
  const status = store.registrationsStatus === 'error' ? notice('Could not load registrations. Check that the Registration service is running, then reload.', true) : store.registrationsStatus !== 'loaded' ? notice('Loading registrations…') : '';
  const tabs = [['pending', 'Waiting'], ['approved', 'Approved'], ['rejected', 'Rejected'], ['all', 'All']];
  if (!tabs.some(([k]) => k === ui.regTab)) ui.regTab = 'pending';
  const q = ui.query.toLowerCase();
  const list = store.registrations.filter((r) => (ui.regTab === 'all' || r.status === ui.regTab) && `${teamName(r.teamId)} ${tournamentName(r.tournamentId)}`.toLowerCase().includes(q))
    .sort((a, b) => (b.submittedAt?.valueOf() || 0) - (a.submittedAt?.valueOf() || 0));
  const rows = list.map((r) => {
    const team = byId(store.teams, r.teamId);
    const manager = admin ? (team?.managerId ? esc(userName(team.managerId) || 'Team manager') : '<span class="chip">Admin-created</span>') : '';
    const decide = admin && r.status === 'pending' ? `${button('Approve', 'approve-registration', { cls: 'btn-sm btn-primary', ic: 'check', data: { id: r.id }, roleAction: 'review-registration' })}${button('Reject', 'reject-registration', { cls: 'btn-sm', ic: 'x', data: { id: r.id }, roleAction: 'review-registration' })}` : '';
    return `<tr><td><div class="who"><span class="crest">${esc(team?.short || initials(teamName(r.teamId)))}</span><span><strong>${esc(teamName(r.teamId))}</strong><small>${esc(team?.dzongkhag || '')}</small></span></div></td><td>${esc(tournamentName(r.tournamentId))}</td>${admin ? `<td>${manager}</td>` : ''}<td>${esc(r.submittedAt ? formatDate(r.submittedAt) : '—')}</td><td>${registrationBadge(r)}</td><td class="row-actions">${decide}</td></tr>`;
  }).join('');
  // Teams with a tournament but no entry (for example teams created before registrations were connected).
  const candidates = (admin ? store.teams : myTeams()).filter((t) => t.tournamentId && !registrationFor(t.id, t.tournamentId));
  const notEntered = candidates.length ? panel('Not entered yet', `${plural(candidates.length, 'team')} linked to a tournament without an entry. Fixtures skip these teams.`, `<div class="table-wrap"><table class="table"><tbody>${candidates.map((t) => `<tr><td><div class="who"><span class="crest">${esc(t.short)}</span><span><strong>${esc(t.name)}</strong><small>${esc(tournamentName(t.tournamentId))}</small></span></div></td><td class="row-actions">${teamActions(t) || '<span class="chip">Registration closed</span>'}</td></tr>`).join('')}</tbody></table></div>`) : '';
  const counts = Object.fromEntries(tabs.map(([k]) => [k, k === 'all' ? store.registrations.length : store.registrations.filter((r) => r.status === k).length]));
  const head = admin
    ? pageHead('Tournament entries', 'Registrations', 'Approve the teams that may play. Fixtures are generated only for approved teams; a rejection needs a reason.', '')
    : pageHead('Tournament entries', 'My registrations', 'The tournaments your teams have entered, and whether the league admin has approved them.', button('Create a team', 'new-team', { cls: 'btn-primary', ic: 'plus' }));
  return `${head}${status}
  <div class="toolbar"><div class="tabs">${tabs.map(([k, l]) => `<button type="button" class="tab ${ui.regTab === k ? 'active' : ''}" data-action="reg-tab" data-tab="${k}">${l}<span class="count">${counts[k]}</span></button>`).join('')}</div><span class="spacer"></span><label class="search">${icon('search')}<input id="page-search" placeholder="Search teams or tournaments…" value="${esc(ui.query)}" /></label></div>
  <section class="panel">${rows ? `<div class="table-wrap"><table class="table"><thead><tr><th>Team</th><th>Tournament</th>${admin ? '<th>Manager</th>' : ''}<th>Entered</th><th>Status</th><th></th></tr></thead><tbody>${rows}</tbody></table></div>` : empty(ui.regTab === 'pending' ? 'Nothing waiting' : 'No entries here', ui.regTab === 'pending' ? (admin ? 'New team entries from managers appear here for your decision.' : 'Entries you submit appear here until the admin decides.') : 'Try another tab or search.', { ic: 'flag' })}</section>
  ${notEntered}`;
}

function reportsPage() {
  const blocked = gate('Reports', 'Insights & exports');
  if (blocked) return blocked;
  const total = store.matches.length;
  const done = store.matches.filter((m) => m.status === 'completed').length;
  const withOfficial = store.matches.filter((m) => m.officialId).length;
  const dz = new Set(store.teams.map((t) => t.dzongkhag).filter(Boolean)).size;
  const exports = [
    ['Standings', `${tournamentName(store.standings.tournamentId)} · ${plural(store.standings.rows.length, 'team')}`, 'export-standings'],
    ['Fixtures and results', `${plural(total, 'match', 'matches')} · all tournaments`, 'export-fixtures'],
    ['Teams', `${plural(store.teams.length, 'team')} with their records`, 'export-teams']
  ];
  return `${pageHead('Insights & exports', 'Reports', 'Season figures from saved data, and CSV downloads.', '')}
  <section class="metrics">${metric('Match completion', total ? `${Math.round((done / total) * 100)}%` : '—', `${done} of ${total} finalised`, 'check')}${metric('Officials assigned', total ? `${Math.round((withOfficial / total) * 100)}%` : '—', `${withOfficial} of ${total} matches`, 'whistle')}${metric('Dzongkhags', `${dz} / 20`, 'represented by teams', 'pin')}${metric('Tournaments', store.tournaments.length, 'this season', 'trophy')}</section>
  ${panel('Downloads', 'CSV files built from the data loaded in this page', `<div class="match-list">${exports.map(([title, sub, action]) => `<div class="check-item"><span class="check-dot">${icon('file')}</span><span><strong>${esc(title)}</strong><small>${esc(sub)}</small></span>${button('Download', action, { cls: 'btn-sm', ic: 'download' })}</div>`).join('')}</div>`)}`;
}

const PAGES = { dashboard: dashboardPage, tournaments: tournamentsPage, tournament: tournamentPage, teams: teamsPage, squads: squadsPage, registrations: registrationsPage, matches: matchesPage, standings: standingsPage, officials: officialsPage, users: usersPage, venues: venuesPage, reports: reportsPage };

/* ---------------------------------------------------------------- chrome (sidebar, top bar) */
function renderChrome() {
  const r = role();
  const allowed = rolePages[r];
  document.querySelectorAll('.nav-link[data-page]').forEach((a) => {
    const show = allowed.includes(a.dataset.page);
    a.hidden = !show;
    a.classList.toggle('active', a.dataset.page === ui.page);
    if (a.dataset.page === ui.page) a.setAttribute('aria-current', 'page'); else a.removeAttribute('aria-current');
  });
  const adminNav = document.getElementById('admin-nav');
  if (adminNav) adminNav.hidden = ![...adminNav.querySelectorAll('.nav-link')].some((a) => !a.hidden);
  const teamLabel = document.querySelector('[data-nav-label="teams"]');
  if (teamLabel) teamLabel.textContent = r === 'team_manager' ? 'My teams' : 'Teams';
  document.getElementById('crumb-current').textContent = PAGE_LABELS[ui.page] || 'Overview';
  document.getElementById('footer-year').textContent = String(new Date().getFullYear());

  const seasons = store.tournaments.map((t) => String(t.season || '')).filter(Boolean)
    .sort((a, b) => a.localeCompare(b, undefined, { numeric: true }));
  document.getElementById('season-label').textContent = `${seasons.at(-1) || new Date().getFullYear()} season`;

  const info = ROLE_INFO[r];
  const card = document.getElementById('role-card');
  const profileButton = document.getElementById('profile-button');
  if (store.session.status === 'checking') {
    card.innerHTML = '<h3>Checking your session…</h3>';
  } else if (!signedIn()) {
    card.innerHTML = `<h3>${icon('eye')}Browsing as a guest</h3><p>Fixtures, results and standings are public. Create a free account to see squads, or sign in to run your part of the season.</p><div class="role-card-actions">${button('Create account', 'signup', { cls: 'btn-light btn-sm', ic: 'userplus' })}${button('Sign in', 'signin', { cls: 'btn-ghost btn-sm', ic: 'login' })}</div>`;
  } else {
    card.innerHTML = `<h3>${icon(info.icon)}You are a ${esc(info.title)}</h3><ul>${info.can.map((c) => `<li>${esc(c)}</li>`).join('')}</ul>`;
  }
  const name = signedIn() ? me().username : 'Not signed in';
  document.getElementById('profile-avatar').textContent = signedIn() ? initials(me().username) : '?';
  document.getElementById('profile-name').textContent = name;
  document.getElementById('profile-role').textContent = signedIn() ? info.title : 'Guest';
  profileButton.innerHTML = icon(signedIn() ? 'logout' : 'login');
  profileButton.dataset.action = signedIn() ? 'signout' : 'signin';
  profileButton.setAttribute('aria-label', signedIn() ? `Sign out ${name}` : 'Sign in');

  const account = document.getElementById('account');
  account.innerHTML = signedIn()
    ? `<button type="button" class="account-button" data-action="account-menu" aria-haspopup="true" aria-expanded="false"><span class="avatar">${esc(initials(me().username))}</span><strong>${esc(me().username)}</strong>${roleBadge(r)}</button>
       <div class="account-menu" id="account-menu" hidden><div class="who"><span class="avatar">${esc(initials(me().username))}</span><span><strong>${esc(me().username)}</strong><small>${esc(info.title)}</small></span></div><p>${esc(info.can.join('. '))}.</p>${button('Sign out', 'signout', { cls: 'btn-sm btn-block', ic: 'logout' })}</div>`
    : store.session.status === 'checking' ? '' : `${button('Create account', 'signup', { cls: 'btn-sm account-signup', ic: 'userplus' })}${button('Sign in', 'signin', { cls: 'btn-primary btn-sm', ic: 'login' })}`;
}

/* ---------------------------------------------------------------- routing */
const landingActive = () => Boolean(window.DrukLanding && window.DrukLanding.active());

function render(page = ui.page, options = {}) {
  const r = role();
  // A tournament page carries its id in the hash: #tournament/<id>.
  if (String(page).startsWith('tournament/')) { ui.tournamentId = String(page).slice('tournament/'.length); page = 'tournament'; }
  if (!PAGES[page]) page = 'dashboard';
  // While the saved session is still being checked, keep the requested page; it re-renders once the role is known.
  if (store.session.status !== 'checking' && !rolePages[r].includes(page)) {
    if (signedIn()) toast(`${ROLE_INFO[r].title} accounts cannot open ${PAGE_LABELS[page] || 'that page'}. Showing your overview.`);
    page = 'dashboard';
  }
  if (page !== ui.page) ui.query = '';
  ui.page = page;
  if (!landingActive()) {
    const hash = page === 'tournament' && ui.tournamentId ? `#tournament/${ui.tournamentId}` : `#${page}`;
    if (options.history !== false && location.hash !== hash) history.pushState({ page }, '', hash);
    else if (options.history === false && location.hash !== hash) history.replaceState({ page }, '', hash);
  }
  renderChrome();
  document.getElementById('page-content').innerHTML = PAGES[page]();
  document.getElementById('sidebar').classList.remove('open');
  if (!landingActive() && options.scroll !== false) window.scrollTo({ top: 0, behavior: 'smooth' });
}
// Re-render the current page in place (data refreshed, filter changed) without moving the scroll position.
function refreshView() { render(ui.page, { history: false, scroll: false }); }

window.addEventListener('hashchange', () => {
  if (landingActive()) return;
  render(location.hash.slice(1) || 'dashboard', { history: false });
});

/* ---------------------------------------------------------------- toasts & modals */
function toast(message, tone = 'info') {
  const kind = ['success', 'error', 'info'].includes(tone) ? tone : 'info';
  const node = document.createElement('div');
  node.className = `toast toast-${kind}`;
  if (kind === 'error') node.setAttribute('role', 'alert');
  node.textContent = message;
  document.getElementById('toast-region').append(node);
  setTimeout(() => node.remove(), kind === 'error' ? 6500 : 3600);
}
function openModal(id, title, sub, body, foot) {
  document.getElementById('modal-root').innerHTML = `<div class="modal-backdrop" data-action="dismiss-modal"><section class="modal" role="dialog" aria-modal="true" aria-labelledby="${id}-title"><header class="modal-head"><div><h2 id="${id}-title">${esc(title)}</h2>${sub ? `<p>${esc(sub)}</p>` : ''}</div><button type="button" class="modal-close" data-action="dismiss-modal" aria-label="Close">×</button></header><form id="${id}" novalidate><div class="modal-body">${body}</div><footer class="modal-foot">${foot}</footer></form></section></div>`;
  document.querySelector('#modal-root input:not([type=hidden]), #modal-root select')?.focus();
}
const closeModal = () => { document.getElementById('modal-root').innerHTML = ''; };
const cancelButton = '<button type="button" class="btn" data-action="dismiss-modal">Cancel</button>';
const options = (list, selected) => list.map((v) => `<option ${v === selected ? 'selected' : ''}>${esc(v)}</option>`).join('');

function showFormError(form, message) {
  let alert = form.querySelector('.form-error');
  if (!alert) { alert = document.createElement('p'); alert.className = 'form-error'; alert.setAttribute('role', 'alert'); form.querySelector('.modal-body, .auth-fields')?.prepend(alert); }
  alert.textContent = message;
}
function setBusy(form, busy, label = 'Saving…') {
  form.setAttribute('aria-busy', String(busy));
  form.querySelectorAll('button[type="submit"]').forEach((b) => {
    if (busy) { b.dataset.label = b.textContent; b.disabled = true; if (b === form._submitter || !form._submitter) b.textContent = label; } else { b.disabled = false; if (b.dataset.label) b.textContent = b.dataset.label; }
  });
}
// Run a save: busy state while it runs; on success close the dialog, confirm, and reload data;
// on failure keep the dialog open with the error inside it.
async function runSave(form, work, success, busyLabel) {
  form.querySelector('.form-error')?.remove();
  setBusy(form, true, busyLabel);
  try {
    const result = await work();
    closeModal();
    toast(typeof success === 'function' ? success(result) : success, 'success');
    await window.DrukAPI.reload();
  } catch (error) {
    setBusy(form, false);
    showFormError(form, error.message || 'Something went wrong.');
  }
}

function openTournamentModal() {
  const year = new Date().getFullYear();
  const seasons = [...new Set([String(year), String(year + 1), ...store.tournaments.map((t) => String(t.season || ''))].filter(Boolean))]
    .sort((a, b) => a.localeCompare(b, undefined, { numeric: true })).reverse();
  openModal('tournament-form', 'Create a tournament', 'Teams can register once it is created.', `<div class="form-grid">
    <div class="field full"><label for="t-name">Tournament name</label><input id="t-name" name="name" maxlength="70" placeholder="e.g. Wangdue Summer Cup" required /></div>
    <div class="field"><label for="t-sport">Sport</label><select id="t-sport" name="sport">${options(SPORTS, 'Football')}</select></div>
    <div class="field"><label for="t-season">Season</label><select id="t-season" name="season">${options(seasons, String(year))}</select></div>
    <div class="field"><label for="t-start">Start date</label><input id="t-start" name="starts" type="date" /><small>Fixtures start on this date if it is in the future.</small></div>
    <div class="field"><label for="t-dz">Dzongkhag</label><select id="t-dz" name="dz">${options(DZONGKHAGS, 'Thimphu')}</select></div></div>`,
  `${cancelButton}<button type="submit" class="btn btn-primary">Create tournament</button>`);
}

function openTeamModal(presetTournament = '') {
  const isManager = role() === 'team_manager';
  const choices = store.tournaments.filter((t) => !isManager || t.status === 'registration_open');
  if (!choices.length) {
    openModal('info-form', 'No tournament to join', '', empty(isManager ? 'No tournament is open for registration' : 'Create a tournament first', isManager ? 'Ask the league admin to open a tournament for registration.' : 'A team is always registered for a tournament.', { ic: 'trophy' }), '<button type="button" class="btn btn-primary" data-action="dismiss-modal">OK</button>');
    return;
  }
  openModal('team-form', isManager ? 'Create your team' : 'Add a team', isManager ? 'The team is linked to your account and entered into the tournament. The league admin then approves the entry.' : 'The team is entered into the tournament and approved straight away.', `<div class="form-grid">
    <div class="field full"><label for="team-name">Team name</label><input id="team-name" name="name" maxlength="70" placeholder="e.g. Wangdue Rovers" required /></div>
    <div class="field"><label for="team-dz">Dzongkhag</label><select id="team-dz" name="dz">${options(DZONGKHAGS, 'Thimphu')}</select></div>
    <div class="field"><label for="team-tournament">Tournament</label><select id="team-tournament" name="tournament">${choices.map((t) => `<option value="${esc(t.id)}" ${t.id === presetTournament ? 'selected' : ''}>${esc(t.name)} · ${esc(t.season || '')}</option>`).join('')}</select>${isManager ? '<small>Only tournaments open for registration are listed.</small>' : ''}</div></div>`,
  `${cancelButton}<button type="submit" class="btn btn-primary">${isManager ? 'Create and enter team' : 'Add team'}</button>`);
}

// Enter an existing team into its tournament. The admin's entries are approved straight away.
async function enterTeam(teamId, tournamentId) {
  const api = window.DrukAPI;
  const team = byId(store.teams, teamId);
  const admin = role() === 'super_admin';
  try {
    const reg = await api.registerTeam(tournamentId, teamId);
    if (admin && reg?.id) await api.reviewRegistration(reg.id, true);
    toast(admin ? `${team?.name || 'Team'} entered and approved.` : `${team?.name || 'Team'} entered. The league admin will review it.`, 'success');
  } catch (error) {
    toast(error.message || 'Could not enter the team.', 'error');
  }
  await api.reload();
}

function openRejectModal(id) {
  const reg = byId(store.registrations, id);
  if (!reg) return;
  openModal('reject-form', 'Reject this entry', `${teamName(reg.teamId)} · ${tournamentName(reg.tournamentId)}`, `<input type="hidden" name="registration" value="${esc(reg.id)}" /><div class="field"><label for="r-reason">Reason</label><textarea id="r-reason" name="reason" rows="3" maxlength="500" placeholder="e.g. Squad list incomplete" required></textarea><small>The team manager needs this to understand the decision. A rejected team cannot enter the same tournament again.</small></div>`,
    `${cancelButton}<button type="submit" class="btn btn-danger">Reject entry</button>`);
}

function openPlayerModal(teamId, playerId = '') {
  const team = byId(store.teams, teamId);
  if (!team) return;
  const player = (store.players[teamId]?.rows || []).find((p) => p.id === playerId);
  openModal('player-form', player ? 'Edit player' : 'Add a player', team.name, `<input type="hidden" name="team" value="${esc(team.id)}" /><input type="hidden" name="player" value="${esc(player?.id || '')}" /><div class="form-grid">
    <div class="field"><label for="p-given">First name</label><input id="p-given" name="given" maxlength="60" value="${esc(player?.given || '')}" required /></div>
    <div class="field"><label for="p-family">Last name</label><input id="p-family" name="family" maxlength="60" value="${esc(player?.family || '')}" /><small>Optional.</small></div>
    <div class="field"><label for="p-shirt">Shirt number</label><input id="p-shirt" name="shirt" type="number" min="1" max="99" inputmode="numeric" value="${esc(player?.shirt || '')}" /><small>1–99, optional.</small></div></div>`,
    `${cancelButton}<button type="submit" class="btn btn-primary">${player ? 'Save player' : 'Add player'}</button>`);
}

function openVenueModal() {
  const needed = store.tournaments.find((t) => t.dzongkhag && !store.venues.some((v) => v.dzongkhag === t.dzongkhag));
  openModal('venue-form', 'Add a venue', 'Generated fixtures are booked at venues in the tournament\'s dzongkhag.', `<div class="form-grid">
    <div class="field full"><label for="v-name">Venue name</label><input id="v-name" name="name" minlength="2" maxlength="120" placeholder="e.g. Changlimithang Stadium" required /></div>
    <div class="field"><label for="v-dz">Dzongkhag</label><select id="v-dz" name="dz">${options(DZONGKHAGS, needed?.dzongkhag || 'Thimphu')}</select></div>
    <div class="field"><label for="v-cap">Capacity</label><input id="v-cap" name="capacity" type="number" min="0" max="100000" inputmode="numeric" placeholder="Optional" /></div></div>`,
    `${cancelButton}<button type="submit" class="btn btn-primary">Add venue</button>`);
}

function openGenerateModal(presetTournament = '') {
  if (!store.tournaments.length) { toast('Create a tournament first.'); return; }
  const pending = pendingRegistrations().length;
  openModal('generate-form', 'Generate fixtures', 'Creates a round-robin schedule for every approved team, at venues in the tournament\'s dzongkhag. Running it again does not duplicate matches.', `<div class="form-grid"><div class="field full"><label for="g-tournament">Tournament</label><select id="g-tournament" name="tournament">${store.tournaments.map((t) => {
    const n = approvedIn(t.id).length;
    const venues = store.venues.filter((v) => v.dzongkhag === t.dzongkhag).length;
    return `<option value="${esc(t.id)}" ${t.id === presetTournament ? 'selected' : ''}>${esc(t.name)} — ${plural(n, 'approved team')}, ${plural(venues, 'venue')}</option>`;
  }).join('')}</select><small>A tournament needs at least two approved teams.</small></div>
  ${pending ? `<div class="field full">${notice(`${plural(pending, 'entry', 'entries')} still waiting for approval will not get fixtures. <button type="button" class="link-btn" data-page="registrations">Review first →</button>`, true)}</div>` : ''}</div>`,
  `${cancelButton}<button type="submit" class="btn btn-primary">Generate fixtures</button>`);
}

function openScoreModal(id) {
  const m = byId(store.matches, id);
  if (!m) return;
  if (m.status === 'completed') { toast(`Final result: ${m.home} ${m.resultHome}–${m.resultAway} ${m.away}.`); return; }
  const score = matchScore(m) || [0, 0];
  const sport = sportOfMatch(m);
  const r = rulesFor(sport);
  // Basketball gets +2 and +3 buttons, so a 98-point game is not 98 taps.
  const quick = (key, label) => (r.quick || []).map((n) => `<button type="button" class="quick-step" data-action="step" data-field="${key}" data-step="${n}" aria-label="${label} plus ${n}">+${n}</button>`).join('');
  const side = (key, name, value, label) => `<div class="score-side"><small>${label} · ${esc(r.unit)}</small><strong>${esc(name)}</strong><div class="stepper"><button type="button" data-action="step" data-field="${key}" data-step="-1" aria-label="${label} score down">−</button><input name="${key}" type="number" min="0" max="${r.max}" inputmode="numeric" value="${esc(value)}" aria-label="${esc(name)} ${esc(r.unit.toLowerCase())}" /><button type="button" data-action="step" data-field="${key}" data-step="1" aria-label="${label} score up">+</button></div>${quick(key, label)}</div>`;
  const rule = r.bestOf ? `Best of ${r.bestOf}: the match ends when one side wins ${Math.floor(r.bestOf / 2) + 1} sets.` : r.draw ? '' : `No draws in ${sportInfo(sport).label.toLowerCase()}.`;
  openModal('score-form', m.status === 'live' ? 'Update the score' : 'Start scoring', `${tournamentName(m.tournamentId)} · ${m.venue || 'Venue TBD'}`, `<input type="hidden" name="match" value="${esc(m.id)}" /><div class="score-grid">${side('home', m.home, score[0], 'HOME')}<span class="score-vs">vs</span>${side('away', m.away, score[1], 'AWAY')}</div><p class="score-note">${sportInfo(sport).emoji} ${esc(sportInfo(sport).label)}${rule ? ` · ${esc(rule)}` : ''} Saving keeps the result open. <b>Finalising locks it</b> and updates the standings.</p>`,
    `${cancelButton}<button type="submit" class="btn" value="save">Save live score</button><button type="submit" class="btn btn-primary" value="finalize">Finalise result</button>`);
}

function openAssignModal(id) {
  const m = byId(store.matches, id);
  if (!m) return;
  if (store.usersStatus !== 'loaded') { toast('The official list is still loading. Try again in a moment.'); return; }
  if (!officials().length) {
    openModal('info-form', 'No match officials yet', '', empty('Create an official account first', 'Officials need an account before they can be assigned to a match.', { ic: 'whistle', action: button('Add official', 'new-user', { cls: 'btn-primary', data: { role: 'match_official' } }) }), cancelButton);
    return;
  }
  openModal('assign-form', m.officialId ? 'Change the official' : 'Assign an official', `${m.home} vs ${m.away} · ${dateParts(m.kickoff).long}`, `<input type="hidden" name="match" value="${esc(m.id)}" /><div class="field"><label for="a-official">Match official</label><select id="a-official" name="official">${officials().map((u) => {
    const load = store.matches.filter((x) => x.officialId === u.id && x.status !== 'completed').length;
    return `<option value="${esc(u.id)}" ${u.id === m.officialId ? 'selected' : ''}>${esc(u.username)} — ${plural(load, 'open match', 'open matches')}</option>`;
  }).join('')}</select><small>Only this official will be able to enter and finalise the score.</small></div>`,
  `${cancelButton}<button type="submit" class="btn btn-primary">Assign official</button>`);
}

function openUserModal(presetRole = 'match_official') {
  const roles = [['match_official', 'Match Official — scores assigned matches'], ['team_manager', 'Team Manager — registers their teams'], ['viewer', 'Viewer — read-only access']];
  openModal('user-form', 'Create an account', 'Share the username and password with the person directly.', `<div class="form-grid">
    <div class="field full"><label for="u-role">Role</label><select id="u-role" name="role">${roles.map(([v, l]) => `<option value="${v}" ${v === presetRole ? 'selected' : ''}>${esc(l)}</option>`).join('')}</select></div>
    <div class="field full"><label for="u-name">Username</label><input id="u-name" name="username" autocomplete="off" pattern="[a-zA-Z0-9._-]{3,40}" maxlength="40" placeholder="e.g. tashi.referee" required /><small>3–40 letters, numbers, dots, dashes or underscores.</small></div>
    <div class="field full"><label for="u-pass">Password</label><div class="password-wrap"><input id="u-pass" name="password" type="password" autocomplete="new-password" minlength="10" maxlength="72" required /><button type="button" data-action="toggle-password">Show</button></div><small>At least 10 characters.</small></div></div>`,
  `${cancelButton}<button type="submit" class="btn btn-primary">Create account</button>`);
}

function openMatchModal() {
  if (!store.tournaments.length) { toast('Create a tournament first.'); return; }
  const first = store.tournaments.find((t) => teamsIn(t.id).length >= 2) || store.tournaments[0];
  const venues = [...new Set([...store.venues.map((v) => v.name), ...store.matches.map((m) => m.venue)].filter(Boolean))];
  const local = new Date(Date.now() + 86400000); local.setMinutes(0, 0, 0);
  const pad = (n) => String(n).padStart(2, '0');
  const value = `${local.getFullYear()}-${pad(local.getMonth() + 1)}-${pad(local.getDate())}T${pad(local.getHours())}:00`;
  openModal('match-form', 'Schedule a match', 'Both teams must be registered in the tournament.', `<div class="form-grid">
    <div class="field full"><label for="m-tournament">Tournament</label><select id="m-tournament" name="tournament">${store.tournaments.map((t) => `<option value="${esc(t.id)}" ${t.id === first.id ? 'selected' : ''}>${esc(t.name)}</option>`).join('')}</select></div>
    <div id="m-teams" class="form-grid full" style="grid-column:1/-1">${matchTeamFields(first.id)}</div>
    <div class="field"><label for="m-kickoff">Kick-off</label><input id="m-kickoff" name="kickoff" type="datetime-local" value="${value}" required /></div>
    <div class="field"><label for="m-venue">Venue</label><input id="m-venue" name="venue" list="m-venues" maxlength="80" placeholder="e.g. Changlimithang Stadium" required /><datalist id="m-venues">${venues.map((v) => `<option value="${esc(v)}"></option>`).join('')}</datalist></div>
    <div class="field full"><label for="m-official">Match official (optional)</label><select id="m-official" name="official"><option value="">Assign later</option>${officials().map((u) => `<option value="${esc(u.id)}">${esc(u.username)}</option>`).join('')}</select></div></div>`,
  `${cancelButton}<button type="submit" class="btn btn-primary">Schedule match</button>`);
}
function matchTeamFields(tournamentId) {
  const teams = teamsIn(tournamentId);
  if (teams.length < 2) return `<div class="field full">${notice('This tournament has fewer than two teams. Add teams before scheduling a match.', true)}</div>`;
  const select = (name, label, pick) => `<div class="field"><label for="m-${name}">${label}</label><select id="m-${name}" name="${name}">${teams.map((t, i) => `<option value="${esc(t.id)}" ${i === pick ? 'selected' : ''}>${esc(t.name)}</option>`).join('')}</select></div>`;
  return `${select('home', 'Home team', 0)}${select('away', 'Away team', 1)}`;
}

/* ---------------------------------------------------------------- sign-in screen */
// The account screen: mode 'signin' (existing account) or 'signup' (new Viewer account).
function openSignIn(mode = 'signin') {
  const screen = document.getElementById('auth-screen');
  const signup = mode === 'signup';
  const roles = Object.values(ROLE_INFO).map((info) => `<div><strong>${esc(info.title)}</strong>${esc(info.can[0])}</div>`).join('');
  const tabs = `<div class="tabs auth-tabs" role="tablist">${[['signin', 'Sign in'], ['signup', 'Create account']].map(([k, l]) => `<button type="button" role="tab" class="tab ${mode === k ? 'active' : ''}" aria-selected="${mode === k}" data-action="auth-mode" data-mode="${k}">${l}</button>`).join('')}</div>`;
  const form = signup
    ? `<div><div class="eyebrow">Free viewer account</div><h1>Join <b>DrukLeague</b></h1></div>
      <p>Follow fixtures, live scores, standings and team squads. Officials and team managers get their accounts from the league admin.</p>
      <form id="signup-form" novalidate><div class="auth-fields">
        <div class="field"><label for="signup-user">Username</label><input id="signup-user" name="username" autocomplete="username" pattern="[a-zA-Z0-9._-]{3,40}" maxlength="40" required /><small>3–40 letters, numbers, dots, dashes or underscores.</small></div>
        <div class="field" style="margin-top:14px"><label for="signup-password">Password</label><div class="password-wrap"><input id="signup-password" name="password" type="password" autocomplete="new-password" minlength="10" maxlength="72" required /><button type="button" data-action="toggle-password">Show</button></div><small>At least 10 characters.</small></div>
        <div class="field" style="margin-top:14px"><label for="signup-confirm">Confirm password</label><input id="signup-confirm" name="confirm" type="password" autocomplete="new-password" maxlength="72" required /></div></div>
        <button type="submit" class="btn btn-primary btn-block">Create account</button>
        <p class="auth-help">Already have an account? <button type="button" class="link-btn" data-action="auth-mode" data-mode="signin">Sign in</button></p>
      </form>`
    : `<div><div class="eyebrow">Welcome back</div><h1>Sign in to <b>DrukLeague</b></h1></div>
      <p>Use your own username and password, or the one your league admin gave you.</p>
      <form id="signin-form" novalidate><div class="auth-fields">
        <div class="field"><label for="login-user">Username</label><input id="login-user" name="username" autocomplete="username" required /></div>
        <div class="field" style="margin-top:14px"><label for="login-password">Password</label><div class="password-wrap"><input id="login-password" name="password" type="password" autocomplete="current-password" required /><button type="button" data-action="toggle-password">Show</button></div></div></div>
        <button type="submit" class="btn btn-primary btn-block">Sign in</button>
        <p class="auth-help">New here? <button type="button" class="link-btn" data-action="auth-mode" data-mode="signup">Create a free viewer account</button>. Officials and team managers get their accounts from the league admin.</p>
      </form>`;
  screen.innerHTML = `<aside class="auth-visual"><img src="./assets/landing/night-match.jpg" alt="" decoding="async" />
      <a class="brand" href="#home" data-action="close-signin"><span class="brand-mark"><img src="./assets/drukleague-logo.png" alt="" /></span><span class="brand-type">DrukLeague<small>Tournament operations</small></span></a>
      <div><h2>One account,<br /><b>the right tools</b> for your role.</h2><div class="auth-roles">${roles}</div></div></aside>
    <div class="auth-panel"><div class="auth-card">
      <div class="auth-top"><button type="button" class="btn btn-sm" data-action="close-signin">← Back</button></div>
      ${tabs}${form}</div></div>`;
  screen.hidden = false;
  document.body.style.overflow = 'hidden';
  document.getElementById(signup ? 'signup-user' : 'login-user').focus();
}
function closeSignIn() {
  const screen = document.getElementById('auth-screen');
  screen.hidden = true;
  screen.innerHTML = '';
  document.body.style.overflow = '';
}

/* ---------------------------------------------------------------- CSV exports */
function downloadCSV(name, rows) {
  const csv = rows.map((row) => row.map((v) => {
    let text = String(v ?? '');
    if (/^[=+\-@]/.test(text)) text = `'${text}`; // stop spreadsheet formula injection
    return `"${text.replaceAll('"', '""')}"`;
  }).join(',')).join('\r\n');
  const link = document.createElement('a');
  link.href = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }));
  link.download = `${name}.csv`;
  link.click();
  URL.revokeObjectURL(link.href);
}
const slug = (s) => String(s || 'drukleague').toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'drukleague';
function exportStandings() {
  const s = store.standings;
  if (!s.rows.length) { toast('There are no standings to download yet.'); return; }
  const ranked = [...s.rows].sort((a, b) => b.pts - a.pts || b.gd - a.gd || b.gf - a.gf);
  downloadCSV(`${slug(tournamentName(s.tournamentId))}-standings`, [['Rank', 'Team', 'Dzongkhag', 'Played', 'Won', 'Drawn', 'Lost', 'Goals for', 'Goals against', 'Goal difference', 'Points'], ...ranked.map((t, i) => [i + 1, t.name, t.dz, t.played, t.w, t.d, t.l, t.gf, t.ga, t.gd, t.pts])]);
  toast('Standings downloaded.', 'success');
}
function exportFixtures() {
  if (!store.matches.length) { toast('There are no matches to download yet.'); return; }
  downloadCSV('drukleague-fixtures', [['Tournament', 'Kick-off', 'Home', 'Away', 'Venue', 'Status', 'Home score', 'Away score', 'Official'], ...sortByKickoff(store.matches).map((m) => { const s = matchScore(m); return [tournamentName(m.tournamentId), m.kickoff ? m.kickoff.toISOString() : '', m.home, m.away, m.venue, m.status, s ? s[0] : '', s ? s[1] : '', m.officialId ? (userName(m.officialId) || 'Assigned') : '']; })]);
  toast('Fixtures downloaded.', 'success');
}
function exportTeams() {
  if (!store.teams.length) { toast('There are no teams to download yet.'); return; }
  downloadCSV('drukleague-teams', [['Team', 'Short code', 'Dzongkhag', 'Tournament', 'Played', 'Won', 'Drawn', 'Lost', 'Goal difference', 'Points'], ...store.teams.map((t) => { const s = teamStats(t.id); return [t.name, t.short, t.dzongkhag, tournamentName(t.tournamentId), s.p, s.w, s.d, s.l, s.gd, s.pts]; })]);
  toast('Teams downloaded.', 'success');
}

/* ---------------------------------------------------------------- events */
const requireRole = (action, message) => { if (canRoleAction(action)) return true; toast(message, 'error'); return false; };

document.addEventListener('click', (event) => {
  if (!event.target.closest('.account')) document.getElementById('account-menu')?.setAttribute('hidden', '');
  const nav = event.target.closest('[data-page]');
  if (nav && !nav.closest('#landing')) {
    event.preventDefault();
    if (nav.dataset.tab) ui.matchTab = nav.dataset.tab;
    if (nav.dataset.team) ui.squadTeam = nav.dataset.team;
    if (nav.dataset.page === 'tournament' && nav.dataset.tournament) {
      if (ui.tournamentId !== nav.dataset.tournament) ui.tTab = 'matches';
      ui.tournamentId = nav.dataset.tournament;
    }
    if (nav.dataset.tournament) {
      if (nav.dataset.page === 'matches') ui.matchTournament = nav.dataset.tournament;
      if (nav.dataset.page === 'standings') void window.DrukAPI.loadStandings(nav.dataset.tournament);
    }
    closeModal();
    render(nav.dataset.page);
    return;
  }
  const target = event.target.closest('[data-action]');
  if (!target || target.closest('#landing')) return;
  const { action } = target.dataset;
  switch (action) {
    case 'signin': closeModal(); openSignIn('signin'); break;
    case 'signup': closeModal(); openSignIn('signup'); break;
    case 'auth-mode': openSignIn(target.dataset.mode); break;
    case 'close-signin': event.preventDefault(); closeSignIn(); if (target.getAttribute('href') === '#home') location.hash = '#home'; break;
    case 'signout': void window.DrukAPI.signOut(); break;
    case 'account-menu': {
      const menu = document.getElementById('account-menu');
      const open = menu.hasAttribute('hidden');
      menu.toggleAttribute('hidden', !open);
      target.setAttribute('aria-expanded', String(open));
      break;
    }
    case 'toggle-menu': document.getElementById('sidebar').classList.toggle('open'); break;
    case 'close-menu': document.getElementById('sidebar').classList.remove('open'); break;
    case 'dismiss-modal': if (event.target === target || target.tagName === 'BUTTON') closeModal(); break;
    case 'reload': void window.DrukAPI.reload(); break;
    case 'reload-standings': void window.DrukAPI.loadStandings(store.standings.tournamentId); break;
    case 'new-tournament': if (requireRole('new-tournament', 'Only the Super Admin can create tournaments.')) openTournamentModal(); break;
    case 'new-team': if (requireRole('new-team', 'Your role cannot register teams.')) openTeamModal(target.dataset.tournament); break;
    case 'generate': if (requireRole('generate-fixtures', 'Only the Super Admin can generate fixtures.')) openGenerateModal(target.dataset.tournament); break;
    case 'new-match': if (requireRole('schedule-match', 'Only the Super Admin can schedule matches.')) openMatchModal(); break;
    case 'assign': if (requireRole('assign-official', 'Only the Super Admin can assign officials.')) openAssignModal(target.dataset.id); break;
    case 'new-user': if (requireRole('manage-users', 'Only the Super Admin can create accounts.')) openUserModal(target.dataset.role); break;
    case 'score': if (requireRole('update-score', 'Only match officials can enter scores.')) openScoreModal(target.dataset.id); break;
    case 'register-team':
      if (requireRole('register-team', 'Your role cannot enter teams.')) {
        target.disabled = true;
        void enterTeam(target.dataset.team, target.dataset.tournament);
      }
      break;
    case 'approve-registration': {
      if (!requireRole('review-registration', 'Only the Super Admin can approve entries.')) break;
      const reg = byId(store.registrations, target.dataset.id);
      target.disabled = true;
      window.DrukAPI.reviewRegistration(target.dataset.id, true)
        .then(() => toast(`${teamName(reg?.teamId)} approved for ${tournamentName(reg?.tournamentId)}.`, 'success'))
        .catch((error) => toast(error.message, 'error'))
        .finally(() => window.DrukAPI.reload());
      break;
    }
    case 'reject-registration': if (requireRole('review-registration', 'Only the Super Admin can reject entries.')) openRejectModal(target.dataset.id); break;
    case 'new-player': case 'edit-player': {
      const team = byId(store.teams, target.dataset.team);
      if (canEditSquad(team)) openPlayerModal(team.id, target.dataset.player); else toast('Only the admin or this team\'s manager can change its squad.', 'error');
      break;
    }
    case 'reload-squad': void window.DrukAPI.loadPlayers(target.dataset.team); break;
    case 'new-venue': if (requireRole('manage-venues', 'Only the Super Admin can add venues.')) openVenueModal(); break;
    case 'reg-tab': ui.regTab = target.dataset.tab; refreshView(); break;
    case 't-tab': ui.tTab = target.dataset.tab; refreshView(); break;
    case 'sport': {
      ui.sport = target.dataset.sport;
      // On the standings page, show a table from the chosen sport straight away.
      const first = store.tournaments.find(inSportT);
      if (ui.page === 'standings' && first && !inSportT(byId(store.tournaments, store.standings.tournamentId) || {})) void window.DrukAPI.loadStandings(first.id);
      refreshView();
      break;
    }
    case 'step': {
      const field = target.closest('form')?.elements[target.dataset.field];
      if (field) field.value = Math.max(0, Math.min(Number(field.max) || 99, (Number(field.value) || 0) + Number(target.dataset.step)));
      break;
    }
    case 'toggle-password': {
      const input = target.parentElement.querySelector('input');
      const show = input.type === 'password';
      input.type = show ? 'text' : 'password';
      target.textContent = show ? 'Hide' : 'Show';
      break;
    }
    case 'match-tab': ui.matchTab = target.dataset.tab; refreshView(); break;
    case 'team-scope': ui.teamScope = target.dataset.scope; refreshView(); break;
    case 'user-role': ui.userRole = target.dataset.role; refreshView(); break;
    case 'export-standings': exportStandings(); break;
    case 'export-fixtures': exportFixtures(); break;
    case 'export-teams': exportTeams(); break;
    default: break;
  }
});

document.addEventListener('submit', (event) => {
  const form = event.target;
  const api = window.DrukAPI;
  if (!form.id || !api) return;
  event.preventDefault();
  form._submitter = event.submitter;
  const f = form.elements;
  switch (form.id) {
    case 'signin-form': {
      const username = f.username.value.trim();
      const password = f.password.value;
      if (!username || !password) { showFormError(form, 'Enter your username and password.'); return; }
      form.querySelector('.form-error')?.remove();
      setBusy(form, true, 'Signing in…');
      api.signIn(username, password).then((user) => {
        closeSignIn();
        toast(`Signed in as ${user.username} (${ROLE_INFO[currentRole()].title}).`, 'success');
        if (landingActive()) location.hash = '#dashboard'; else render('dashboard');
      }).catch((error) => { setBusy(form, false); showFormError(form, error.message === 'Request failed (401)' ? 'Wrong username or password.' : error.message); });
      return;
    }
    case 'tournament-form': {
      const name = f.name.value.trim();
      if (!name) { showFormError(form, 'Enter a tournament name.'); return; }
      void runSave(form, () => api.createTournament({ name, sport: f.sport.value.toLowerCase(), season: f.season.value, starts_at: f.starts.value, dzongkhag: f.dz.value }), `${name} created. Teams can now register.`);
      return;
    }
    case 'team-form': {
      const name = f.name.value.trim();
      if (!name) { showFormError(form, 'Enter a team name.'); return; }
      const tournament = byId(store.tournaments, f.tournament.value);
      void runSave(form, () => api.createTeam({ name, short_code: initials(name), dzongkhag: f.dz.value, tournament_id: f.tournament.value }), (result) => {
        const where = tournament?.name || 'the tournament';
        if (result?.registration_status === 'approved') return `${name} added and approved for ${where}.`;
        if (result?.registration_status === 'pending') return `${name} created and entered into ${where}. The league admin will review the entry.`;
        // The team was saved, but the entry failed; say why so it can be entered later.
        setTimeout(() => toast(`${name} was saved but not entered: ${result?.registration_error || 'the entry failed'}.`, 'error'), 0);
        return `${name} saved.`;
      });
      return;
    }
    case 'signup-form': {
      const username = f.username.value.trim();
      if (!/^[a-zA-Z0-9._-]{3,40}$/.test(username)) { showFormError(form, 'Usernames are 3–40 letters, numbers, dots, dashes or underscores.'); return; }
      if (f.password.value.length < 10) { showFormError(form, 'The password must be at least 10 characters.'); return; }
      if (f.password.value !== f.confirm.value) { showFormError(form, 'The two passwords do not match.'); return; }
      form.querySelector('.form-error')?.remove();
      setBusy(form, true, 'Creating account…');
      api.signUp(username, f.password.value).then((user) => {
        closeSignIn();
        toast(`Welcome, ${user.username}! Your viewer account is ready.`, 'success');
        if (landingActive()) location.hash = '#dashboard'; else render('dashboard');
      }).catch((error) => { setBusy(form, false); showFormError(form, error.message); });
      return;
    }
    case 'reject-form': {
      const reason = f.reason.value.trim();
      if (!reason) { showFormError(form, 'Give a reason for the team manager.'); return; }
      const reg = byId(store.registrations, f.registration.value);
      void runSave(form, () => api.reviewRegistration(f.registration.value, false, reason), `${teamName(reg?.teamId)} rejected for ${tournamentName(reg?.tournamentId)}.`);
      return;
    }
    case 'player-form': {
      const given = f.given.value.trim();
      const shirt = f.shirt.value.trim();
      if (!given) { showFormError(form, 'Enter the player\'s first name.'); return; }
      if (shirt && !(Number.isInteger(Number(shirt)) && Number(shirt) >= 1 && Number(shirt) <= 99)) { showFormError(form, 'Shirt numbers run from 1 to 99.'); return; }
      const body = { given_name: given, family_name: f.family.value.trim(), shirt_number: shirt };
      const teamId = f.team.value;
      const playerId = f.player.value;
      void runSave(form, () => (playerId ? api.updatePlayer(playerId, body) : api.createPlayer(teamId, body)), `${given} ${playerId ? 'updated' : 'added to the squad'}.`)
        .then(() => api.loadPlayers(teamId));
      return;
    }
    case 'venue-form': {
      const name = f.name.value.trim();
      if (name.length < 2) { showFormError(form, 'Enter the venue name.'); return; }
      const capacity = f.capacity.value === '' ? 0 : Number(f.capacity.value);
      if (!Number.isInteger(capacity) || capacity < 0 || capacity > 100000) { showFormError(form, 'Capacity must be a whole number up to 100,000.'); return; }
      void runSave(form, () => api.createVenue({ name, dzongkhag: f.dz.value, capacity }), `${name} added in ${f.dz.value}.`);
      return;
    }
    case 'generate-form': {
      const tournament = byId(store.tournaments, f.tournament.value);
      void runSave(form, () => api.generateFixtures(f.tournament.value), (result) => {
        const created = Number(result?.fixtures_created) || 0;
        return created ? `${plural(created, 'fixture')} created for ${tournament?.name}. Now assign officials.` : `No new fixtures for ${tournament?.name}: it already has fixtures or fewer than two teams.`;
      }, 'Generating…');
      return;
    }
    case 'score-form': {
      const home = Number(f.home.value);
      const away = Number(f.away.value);
      const m = byId(store.matches, f.match.value);
      const final = event.submitter?.value === 'finalize';
      const problem = scoreProblem(sportOfMatch(m), home, away, final);
      if (problem) { showFormError(form, problem); return; }
      if (final && !window.confirm(`Finalise ${m.home} ${home}–${away} ${m.away}? The result will be locked.`)) return;
      void runSave(form, () => (final ? api.finalize(m.id, home, away) : api.saveLive(m.id, home, away)),
        final ? `Result finalised: ${m.home} ${home}–${away} ${m.away}. Standings are updating.` : `Live score saved: ${m.home} ${home}–${away} ${m.away}.`, final ? 'Finalising…' : 'Saving…');
      return;
    }
    case 'assign-form': {
      const official = byId(store.users, f.official.value);
      void runSave(form, () => api.assignOfficial(f.match.value, f.official.value), `${official?.username || 'Official'} assigned.`);
      return;
    }
    case 'user-form': {
      const username = f.username.value.trim();
      if (!/^[a-zA-Z0-9._-]{3,40}$/.test(username)) { showFormError(form, 'Usernames are 3–40 letters, numbers, dots, dashes or underscores.'); return; }
      if (f.password.value.length < 10) { showFormError(form, 'The password must be at least 10 characters.'); return; }
      void runSave(form, () => api.createUser({ username, password: f.password.value, role: f.role.value }), `Account ${username} created as ${roleLabels[f.role.value]}.`);
      return;
    }
    case 'match-form': {
      if (!f.home || !f.away) { showFormError(form, 'This tournament needs at least two teams.'); return; }
      if (f.home.value === f.away.value) { showFormError(form, 'Choose two different teams.'); return; }
      const kickoff = new Date(f.kickoff.value);
      if (Number.isNaN(kickoff.valueOf())) { showFormError(form, 'Choose a kick-off date and time.'); return; }
      const venue = f.venue.value.trim();
      if (!venue) { showFormError(form, 'Enter a venue.'); return; }
      void runSave(form, () => api.createMatch({ tournament_id: f.tournament.value, home_team_id: f.home.value, away_team_id: f.away.value, venue, kickoff: kickoff.toISOString(), official_id: f.official.value }), 'Match scheduled.');
      return;
    }
    default:
      closeModal();
  }
});

document.addEventListener('change', (event) => {
  const { id, value } = event.target;
  if (id === 'season-filter') { ui.season = value; refreshView(); }
  if (id === 'match-tournament') { ui.matchTournament = value; refreshView(); }
  if (id === 'standings-tournament') void window.DrukAPI.loadStandings(value);
  if (id === 'm-tournament') document.getElementById('m-teams').innerHTML = matchTeamFields(value);
  if (id === 'squad-team') { ui.squadTeam = value; ui.query = ''; refreshView(); }
});

document.addEventListener('input', (event) => {
  if (event.target.id !== 'page-search') return;
  ui.query = event.target.value;
  const caret = event.target.selectionStart;
  document.getElementById('page-content').innerHTML = PAGES[ui.page]();
  const input = document.getElementById('page-search');
  if (input) { input.focus(); input.setSelectionRange(caret, caret); }
});

document.addEventListener('keydown', (event) => {
  if (event.key === 'Enter' && event.target.matches?.('.t-card-link')) { event.target.click(); return; }
  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') { event.preventDefault(); document.getElementById('global-search')?.focus(); }
  if (event.key === 'Escape') {
    if (!document.getElementById('auth-screen').hidden) closeSignIn();
    closeModal();
    document.getElementById('sidebar').classList.remove('open');
    document.getElementById('account-menu')?.setAttribute('hidden', '');
  }
  if (event.key === 'Enter' && event.target.id === 'global-search') {
    const term = event.target.value.trim().toLowerCase();
    if (!term) return;
    const allowed = rolePages[role()];
    const has = (list, fields) => list.some((x) => fields(x).toLowerCase().includes(term));
    let page = 'matches';
    if (allowed.includes('teams') && has(store.teams, (t) => `${t.name} ${t.dzongkhag}`)) page = 'teams';
    else if (allowed.includes('tournaments') && has(store.tournaments, (t) => t.name)) page = 'tournaments';
    render(page);
    ui.query = event.target.value.trim();
    refreshView();
    event.target.value = '';
  }
});

window.DrukApp = Object.freeze({ render, refreshView, openSignIn, closeSignIn, toast, store, ui });
render(landingActive() ? 'dashboard' : (location.hash.slice(1) || 'dashboard'), { history: false });
