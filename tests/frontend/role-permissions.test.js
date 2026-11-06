const assert = require('node:assert/strict');
const {after, test} = require('node:test');
const {canRoleAction, currentRole, roleActions, roleLabels, rolePages, roleLabel, roleIdentity} = require('../../role-permissions.js');

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
  assert.deepEqual(rolePages.super_admin, ['dashboard','tournaments','teams','matches','standings','officials','venues','reports']);
  assert.deepEqual(rolePages.match_official, ['dashboard','matches','standings']);
  assert.deepEqual(rolePages.team_manager, ['dashboard','teams','matches','standings']);
  assert.deepEqual(rolePages.viewer, ['dashboard','tournaments','teams','matches','standings']);
});

test('unknown and removed roles fall back to Viewer', () => {
  useRole(null);
  assert.equal(currentRole(), 'viewer');
  useRole('dzongkhag_admin');
  assert.equal(currentRole(), 'viewer');
  useRole('unexpected');
  assert.equal(currentRole(), 'viewer');
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

test('admin, team manager, and match official controls are limited to their roles', () => {
  const cases = [
    ['super_admin', ['new-tournament','generate-fixtures','new-team','manage-officials','manage-venues','export']],
    ['team_manager', ['new-team','export']],
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
