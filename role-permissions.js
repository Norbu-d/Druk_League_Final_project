const rolePages = Object.freeze({
  super_admin: Object.freeze(['dashboard','tournaments','teams','matches','standings','officials','venues','reports']),
  match_official: Object.freeze(['dashboard','matches','standings']),
  team_manager: Object.freeze(['dashboard','teams','matches','standings']),
  viewer: Object.freeze(['dashboard','tournaments','teams','matches','standings'])
});

const roleActions = Object.freeze({
  'new-tournament': Object.freeze(['super_admin']),
  'generate-fixtures': Object.freeze(['super_admin']),
  'new-team': Object.freeze(['super_admin','team_manager']),
  'update-score': Object.freeze(['match_official']),
  'finalize-score': Object.freeze(['match_official']),
  'manage-officials': Object.freeze(['super_admin']),
  'manage-venues': Object.freeze(['super_admin']),
  export: Object.freeze(['super_admin','match_official','team_manager','viewer'])
});

const roleLabels = Object.freeze({
  super_admin: 'Super Admin',
  match_official: 'Match Official',
  team_manager: 'Team Manager',
  viewer: 'Viewer'
});

function currentRole() {
  const role = globalThis.sessionStorage?.getItem('drukleague-role') || 'viewer';
  return Object.prototype.hasOwnProperty.call(rolePages, role) ? role : 'viewer';
}

function sanitizeRole(role) {
  return typeof role === 'string' && Object.prototype.hasOwnProperty.call(rolePages, role) ? role : 'viewer';
}

function sanitizeUsername(username) {
  return typeof username === 'string' && /^[a-zA-Z0-9._-]{3,40}$/.test(username) ? username : '';
}

function sanitizeUUID(id) {
  return typeof id === 'string' && /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(id) ? id : '';
}

function safeAPIURL(apiRoot, path, method = 'GET') {
  try {
    const root = new URL(apiRoot);
    if (!['http:','https:'].includes(root.protocol) || root.pathname.replace(/\/+$/, '') !== '/api' || root.search || root.hash) return '';
    const target = new URL(String(path).replace(/^\/+/, ''), `${root.href.replace(/\/+$/, '')}/`);
    const route = target.pathname.slice('/api'.length);
    const id = '[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}';
    const validRoute = new RegExp(`^/(?:auth/(?:login|logout|session)|teams|tournaments|matches|standings|tournaments/${id}/fixtures|matches/${id}/(?:finalize|live))$`, 'i').test(route);
    const verb = String(method || 'GET').toUpperCase();
    const validMethod = route === '/auth/login' || route === '/auth/logout' ? verb === 'POST'
      : route === '/auth/session' || route === '/matches' || route === '/standings' ? verb === 'GET'
      : route === '/teams' || route === '/tournaments' ? ['GET','POST'].includes(verb)
      : route.endsWith('/fixtures') || route.endsWith('/finalize') ? verb === 'POST'
      : route.endsWith('/live') ? verb === 'PATCH' : false;
    const validQuery = target.search === '' || (route === '/standings' && target.searchParams.size === 1 && sanitizeUUID(target.searchParams.get('tournament_id')) !== '');
    if (target.origin !== root.origin || !target.pathname.startsWith('/api/') || !validRoute || !validMethod || !validQuery) return '';
    return target.href;
  } catch (_) { return ''; }
}

function canRoleAction(action) {
  return (roleActions[action] || []).includes(currentRole());
}

function roleLabel(role = currentRole()) {
  return roleLabels[role] || roleLabels.viewer;
}

function roleIdentity(username, role, signedIn = true) {
  const displayName = signedIn ? (username || 'Signed-in user') : 'Public viewer';
  const safeRole = signedIn && Object.prototype.hasOwnProperty.call(roleLabels, role) ? role : 'viewer';
  const badge = displayName.split(/\s+/).map(part => part[0] || '').join('').slice(0, 2).toUpperCase() || 'PV';
  return Object.freeze({displayName, role: safeRole, roleLabel: roleLabels[safeRole], badge});
}

const permissions = Object.freeze({rolePages, roleActions, roleLabels, currentRole, canRoleAction, roleLabel, roleIdentity, sanitizeRole, sanitizeUsername, sanitizeUUID, safeAPIURL});
if (typeof module !== 'undefined' && module.exports) module.exports = permissions;
if (typeof window !== 'undefined') window.DrukLeaguePermissions = permissions;
