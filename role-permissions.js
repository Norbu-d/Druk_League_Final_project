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

function canRoleAction(action) {
  return (roleActions[action] || []).includes(currentRole());
}

function roleLabel(role = currentRole()) {
  return roleLabels[role] || roleLabels.viewer;
}

const permissions = Object.freeze({rolePages, roleActions, roleLabels, currentRole, canRoleAction, roleLabel});
if (typeof module !== 'undefined' && module.exports) module.exports = permissions;
if (typeof window !== 'undefined') window.DrukLeaguePermissions = permissions;
