const assert = require('node:assert/strict');
const {after, test} = require('node:test');
const {canRoleAction, currentRole, roleActions, roleLabels, rolePages, roleLabel, roleIdentity, sanitizeRole, sanitizeUsername, sanitizeUUID, safeAPIURL} = require('../../role-permissions.js');

const originalSessionStorage = globalThis.sessionStorage;
after(() => {
  if (originalSessionStorage === undefined) delete globalThis.sessionStorage;
  else globalThis.sessionStorage = originalSessionStorage;
});

function useRole(role) {
  globalThis.sessionStorage = {getItem: () => role};
}

test('only the four supported roles are defined, with the expected page access', () => {
  assert.deepEqual(Object.keys(rolePages).sort(), ['match_official','super_admin','team_manager','viewer']);
  assert.deepEqual(rolePages.super_admin, ['dashboard','tournaments','tournament','teams','squads','registrations','matches','standings','officials','users','venues','reports']);
  assert.deepEqual(rolePages.match_official, ['dashboard','tournaments','tournament','matches','standings']);
  assert.deepEqual(rolePages.team_manager, ['dashboard','tournaments','tournament','teams','squads','registrations','matches','standings']);
  assert.deepEqual(rolePages.viewer, ['dashboard','tournaments','tournament','teams','squads','matches','standings']);
});

test('unknown and removed roles fall back to Viewer', () => {
  useRole(null);
  assert.equal(currentRole(), 'viewer');
  useRole('dzongkhag_admin');
  assert.equal(currentRole(), 'viewer');
  useRole('unexpected');
  assert.equal(currentRole(), 'viewer');
});

test('API identity values are normalized before reaching browser storage', () => {
  assert.equal(sanitizeRole('team_manager'), 'team_manager');
  assert.equal(sanitizeRole('administrator<script>'), 'viewer');
  assert.equal(sanitizeUsername('norbu.dhendup-1'), 'norbu.dhendup-1');
  assert.equal(sanitizeUsername('<img src=x onerror=alert(1)>'), '');
  assert.equal(sanitizeUUID('22222222-2222-4222-8222-222222222222'), '22222222-2222-4222-8222-222222222222');
  assert.equal(sanitizeUUID('../admin'), '');
});

test('API requests are confined to known same-origin routes and methods', () => {
  const root = 'http://localhost/api';
  assert.equal(safeAPIURL(root, '/teams'), 'http://localhost/api/teams');
  assert.equal(safeAPIURL(root, '/matches/22222222-2222-4222-8222-222222222222/finalize', 'POST'), 'http://localhost/api/matches/22222222-2222-4222-8222-222222222222/finalize');
  assert.equal(safeAPIURL(root, '/standings?tournament_id=22222222-2222-4222-8222-222222222222'), 'http://localhost/api/standings?tournament_id=22222222-2222-4222-8222-222222222222');
  assert.equal(safeAPIURL(root, '//attacker.example/collect'), '');
  assert.equal(safeAPIURL(root, '/../admin'), '');
  assert.equal(safeAPIURL(root, '/matches/not-a-uuid/live', 'PATCH'), '');
  assert.equal(safeAPIURL(root, '/auth/login'), '');
  assert.equal(safeAPIURL(root, '/standings?tournament_id=invalid&next=/admin'), '');
});

test('admin account, scheduling and session-refresh routes are allowed with their methods only', () => {
  const root = 'http://localhost/api';
  const id = '22222222-2222-4222-8222-222222222222';
  assert.equal(safeAPIURL(root, '/auth/refresh', 'POST'), 'http://localhost/api/auth/refresh');
  assert.equal(safeAPIURL(root, '/auth/refresh', 'GET'), '');
  assert.equal(safeAPIURL(root, '/users'), 'http://localhost/api/users');
  assert.equal(safeAPIURL(root, '/users', 'POST'), 'http://localhost/api/users');
  assert.equal(safeAPIURL(root, '/users?role=match_official'), 'http://localhost/api/users?role=match_official');
  assert.equal(safeAPIURL(root, '/users?role=admin'), '');
  assert.equal(safeAPIURL(root, '/users?role=match_official', 'POST'), '');
  assert.equal(safeAPIURL(root, '/users', 'DELETE'), '');
  assert.equal(safeAPIURL(root, '/matches', 'POST'), 'http://localhost/api/matches');
  assert.equal(safeAPIURL(root, `/matches/${id}/official`, 'PATCH'), `http://localhost/api/matches/${id}/official`);
  assert.equal(safeAPIURL(root, `/matches/${id}/official`, 'POST'), '');
});

test('the active role has a clear user-facing label', () => {
  assert.deepEqual(roleLabels, {
    super_admin: 'Super Admin',
    match_official: 'Match Official',
    team_manager: 'Team Manager',
    viewer: 'Viewer'
  });
  for (const [role, label] of Object.entries(roleLabels)) {
    useRole(role);
    assert.equal(roleLabel(), label);
  }
  assert.equal(roleLabel('unknown'), 'Viewer');
});

test('the profile identity reflects signed-in role and resets cleanly when signed out', () => {
  assert.deepEqual(roleIdentity('Namgay Lhamo', 'super_admin'), {
    displayName: 'Namgay Lhamo', role: 'super_admin', roleLabel: 'Super Admin', badge: 'NL'
  });
  assert.deepEqual(roleIdentity('', 'unexpected'), {
    displayName: 'Signed-in user', role: 'viewer', roleLabel: 'Viewer', badge: 'SU'
  });
  assert.deepEqual(roleIdentity('Namgay Lhamo', 'super_admin', false), {
    displayName: 'Public viewer', role: 'viewer', roleLabel: 'Viewer', badge: 'PV'
  });
});

test('public views, sign-up, registrations, squads and venues are allowed with their methods only', () => {
  const root = 'http://localhost/api';
  const id = '22222222-2222-4222-8222-222222222222';
  for (const name of ['tournaments','teams','matches']) {
    assert.equal(safeAPIURL(root, `/public/${name}`), `http://localhost/api/public/${name}`);
    assert.equal(safeAPIURL(root, `/public/${name}`, 'POST'), '');
  }
  assert.equal(safeAPIURL(root, `/public/standings?tournament_id=${id}`), `http://localhost/api/public/standings?tournament_id=${id}`);
  assert.equal(safeAPIURL(root, '/public/standings?tournament_id=x'), '');
  assert.equal(safeAPIURL(root, '/public/users'), '');
  assert.equal(safeAPIURL(root, '/auth/signup', 'POST'), 'http://localhost/api/auth/signup');
  assert.equal(safeAPIURL(root, '/auth/signup'), '');
  assert.equal(safeAPIURL(root, '/registrations'), 'http://localhost/api/registrations');
  assert.equal(safeAPIURL(root, '/registrations', 'POST'), 'http://localhost/api/registrations');
  assert.equal(safeAPIURL(root, `/registrations?tournament_id=${id}&status=pending`), `http://localhost/api/registrations?tournament_id=${id}&status=pending`);
  assert.equal(safeAPIURL(root, '/registrations?status=deleted'), '');
  assert.equal(safeAPIURL(root, '/registrations?status=pending&status=approved'), '');
  assert.equal(safeAPIURL(root, `/registrations/${id}`, 'PATCH'), `http://localhost/api/registrations/${id}`);
  assert.equal(safeAPIURL(root, `/registrations/${id}`, 'DELETE'), '');
  assert.equal(safeAPIURL(root, '/registrations/not-a-uuid', 'PATCH'), '');
  assert.equal(safeAPIURL(root, `/teams/${id}/players`), `http://localhost/api/teams/${id}/players`);
  assert.equal(safeAPIURL(root, `/teams/${id}/players`, 'POST'), `http://localhost/api/teams/${id}/players`);
  assert.equal(safeAPIURL(root, `/players/${id}`, 'PATCH'), `http://localhost/api/players/${id}`);
  assert.equal(safeAPIURL(root, `/players/${id}`), '');
  assert.equal(safeAPIURL(root, '/venues'), 'http://localhost/api/venues');
  assert.equal(safeAPIURL(root, '/venues', 'POST'), 'http://localhost/api/venues');
  assert.equal(safeAPIURL(root, '/venues', 'PATCH'), '');
});

test('admin, team manager, and match official controls are limited to their roles', () => {
  const cases = [
    ['super_admin', ['new-tournament','generate-fixtures','new-team','register-team','review-registration','edit-squad','manage-officials','manage-venues','manage-users','schedule-match','assign-official','export']],
    ['team_manager', ['new-team','register-team','edit-squad','export']],
    ['match_official', ['update-score','finalize-score','export']],
    ['viewer', ['export']]
  ];
  for (const [role, allowed] of cases) {
    useRole(role);
    for (const action of Object.keys(roleActions)) {
      assert.equal(canRoleAction(action), allowed.includes(action), `${role} permission for ${action}`);
    }
  }
});

test('all supported roles may export read-only data; unknown actions are denied', () => {
  for (const role of Object.keys(rolePages)) {
    useRole(role);
    assert.equal(canRoleAction('export'), true, `${role} can export read-only data`);
    assert.equal(canRoleAction('delete-user'), false, `${role} cannot use an undefined action`);
  }
});
