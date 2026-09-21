const test = require('node:test');
const assert = require('node:assert/strict');
const {spawnSync} = require('node:child_process');
const {annotation, argumentsFor} = require('./run.cjs');
test('annotations escape workflow command injection', () => {
  const text = annotation({severity: 'error', file: 'a,b.go', start_line: 0, end_line: 3, rule: 'x', message: 'bad\n::error::injected%'});
  assert.equal(text, '::error file=a%2Cb.go,line=1,endLine=3,title=x::bad%0A::error::injected%25');
});
test('inputs are separate arguments and paths follow separator', () => {
  assert.deepEqual(argumentsFor({REAPER_ACTION_TASK: '$(bad)', REAPER_ACTION_PATHS: 'a b\n--debug'}, {pull_request: {base: {sha: 'abc'}}}), ['check', '--format', 'json', '--diff', 'abc', '--verbose', '--task', '$(bad)', '--', 'a b', '--debug']);
  assert.ok(argumentsFor({}, {pull_request: {base: {sha: 'abc'}}}).includes('--task-from-pr'));
  assert.throws(() => argumentsFor({}, {}), /base/);
});
test('missing fork secrets fail before invoking a provider', () => {
  const result = spawnSync(process.execPath, [require.resolve('./run.cjs')], {env: {...process.env, REAPER_ACTION_API_KEY: ''}, encoding: 'utf8'});
  assert.equal(result.status, 2);
  assert.match(result.stdout, /secret unavailable/);
});
