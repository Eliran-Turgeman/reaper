const fs = require('node:fs');
const path = require('node:path');
const {spawnSync} = require('node:child_process');

function escapeData(value) {
  return String(value).replaceAll('%', '%25').replaceAll('\r', '%0D').replaceAll('\n', '%0A');
}
function escapeProperty(value) {
  return escapeData(value).replaceAll(':', '%3A').replaceAll(',', '%2C');
}
function annotation(item) {
  const level = item.severity === 'error' ? 'error' : 'warning';
  const location = item.file === '<patch>' ? '' : `file=${escapeProperty(item.file)},line=${Math.max(1, item.start_line)},endLine=${Math.max(1, item.start_line, item.end_line)},`;
  return `::${level} ${location}title=${escapeProperty(item.rule)}::${escapeData(item.message)}`;
}
function argumentsFor(env, event) {
  const base = env.REAPER_ACTION_BASE || event.pull_request?.base?.sha;
  if (!base || base.startsWith('-')) throw new Error('A pull request base or explicit base input is required.');
  const args = ['check', '--format', 'json', '--diff', base];
  if (!env.REAPER_ACTION_TASK) args.push('--task-from-pr');
  args.push('--verbose');
  for (const [input, flag] of [['TASK', 'task'], ['PROVIDER', 'provider'], ['MODEL', 'model'], ['CONFIG', 'config']]) {
    if (env[`REAPER_ACTION_${input}`]) args.push(`--${flag}`, env[`REAPER_ACTION_${input}`]);
  }
  if (env.REAPER_ACTION_FAIL_ON_WARNING === 'true') args.push('--fail-on-warning');
  args.push('--', ...(env.REAPER_ACTION_PATHS || '').split(/\r?\n/).map(x => x.trim()).filter(Boolean));
  return args;
}
function main() {
  const env = process.env;
  if (!env.REAPER_ACTION_API_KEY) throw new Error('Provider secret unavailable. Fork PRs cannot run semantic analysis; run in a trusted context with credentials.');
  const event = JSON.parse(fs.readFileSync(env.GITHUB_EVENT_PATH, 'utf8'));
  const provider = env.REAPER_ACTION_PROVIDER || 'typesafe';
  if (!['typesafe', 'openrouter'].includes(provider)) throw new Error('Unknown provider');
  const childEnv = {...env, [provider === 'openrouter' ? 'OPENROUTER_API_KEY' : 'TYPESAFE_API_KEY']: env.REAPER_ACTION_API_KEY};
  const binary = path.join(env.RUNNER_TEMP, 'reaper-action-bin', process.platform === 'win32' ? 'reaper.exe' : 'reaper');
  const result = spawnSync(binary, argumentsFor(env, event), {encoding: 'utf8', env: childEnv, maxBuffer: 32 * 1024 * 1024});
  if (result.error) throw result.error;
  // Provider output is untrusted and must never become a workflow command.
  if (result.stderr) console.log(escapeData(result.stderr));
  if (result.stdout) {
    const report = JSON.parse(result.stdout);
    for (const item of report.diagnostics) console.log(annotation(item));
    if (!report.complete) console.log(`::error::${escapeData('Reaper analysis incomplete; inspect skipped_units in the report.')}`);
    fs.writeFileSync(path.join(env.RUNNER_TEMP, 'reaper-report.json'), result.stdout);
  }
  process.exitCode = result.status ?? 2;
}
module.exports = {escapeData, annotation, argumentsFor};
if (require.main === module) {
  try { main(); } catch (error) { console.log(`::error::${escapeData(error.message)}`); process.exitCode = 2; }
}
