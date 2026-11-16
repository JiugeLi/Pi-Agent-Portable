"""Check real executable project paths, legacy imports and native session reopening without model calls."""
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess

parser = argparse.ArgumentParser()
parser.add_argument('--release-dir', type=Path, required=True)
parser.add_argument('--work-dir', type=Path, required=True)
parser.add_argument('--platform', choices=['windows', 'linux'], required=True)
parser.add_argument('--reuse', action='store_true', help='Repeat checks in an existing fixture directory')
args = parser.parse_args()
release, root = args.release_dir.resolve(), args.work_dir.resolve()
if root.exists() and not args.reuse:
    raise RuntimeError('Use a fresh project verification directory.')
app, first, second = root / 'app 中文', root / 'launch one', root / 'launch two'
for directory in (app, first, second):
    directory.mkdir(parents=True, exist_ok=True)
exe = app / ('pi-windows-x64.exe' if args.platform == 'windows' else 'pi-linux-x64')
shutil.copy2(release / exe.name, exe)
if args.platform == 'linux':
    exe.chmod(0o700)
env = {k: v for k, v in os.environ.items() if k not in (
    'PI_PORTABLE_PROJECT', 'PI_PORTABLE_CONFIG', 'PI_CODING_AGENT_DIR', 'PI_PACKAGE_DIR',
    'PI_CODING_AGENT_SESSION_DIR', 'SOLOMUX_API_KEY', 'SOLOAPI_API_KEY', 'PI_PORTABLE_API_KEY', 'NODE_OPTIONS', 'NODE_PATH')}

def run(options, launch=first, records=None):
    kwargs = {'stdin': subprocess.DEVNULL} if records is None else {'input': records}
    result = subprocess.run([str(exe), *options], cwd=launch, env=env,
        capture_output=True, text=True, encoding='utf-8', errors='replace', timeout=240, **kwargs)
    assert result.returncode == 0, f'Project check failed with exit {result.returncode}'
    return result

print('Checking project paths from an unrelated folder...', flush=True)
info = json.loads(run(['--portable-diagnostics']).stdout)
project = app / 'pi-project'
assert Path(info['cwd']) == project and Path(info['projectDir']) == project
assert Path(info['configFile']) == project / 'config.json'
assert not (app / 'config.json').exists()
if not args.reuse:
    assert not (first / '.pi-portable').exists()
assert (project / '.gitignore').read_text(encoding='utf-8') == '.pi-portable/\n/config.json\n'
info_file = root / 'info.json'
info_file.write_text(json.dumps(info), encoding='utf-8')
node = Path(info['runtime']) / ('node/node.exe' if args.platform == 'windows' else 'node/bin/node')
helper = Path(__file__).with_name('verify_project_sessions.mjs')

def inspect(mode):
    result = subprocess.run([str(node), str(helper), mode, str(info_file), str(first)],
        cwd=project, env=env, stdin=subprocess.DEVNULL, capture_output=True,
        text=True, encoding='utf-8', errors='replace', timeout=60)
    assert result.returncode == 0, 'Native session verification failed: ' + result.stderr[:2000]
    print(result.stdout.strip(), flush=True)

if not (root / 'project-fixtures.json').exists():
    inspect('create')
print('Checking import, repeated launch, config edits and a different launch directory...', flush=True)
assert json.loads(run(['--portable-diagnostics']).stdout)['projectDir'] == str(project)
config = project / 'config.json'
original = config.read_bytes()
edited = json.loads(original)
edited['defaultModel'] = 'PROJECT_CONFIG_CHECK'
config.write_text(json.dumps(edited), encoding='utf-8')
warm = run(['--portable-diagnostics'], second)
assert not warm.stderr.strip()
assert json.loads(warm.stdout)['model'] == 'PROJECT_CONFIG_CHECK'
assert config.read_text(encoding='utf-8') == json.dumps(edited)
config.write_bytes(original)
env['PI_PORTABLE_PROJECT'] = str(project)
assert json.loads(run(['--portable-diagnostics'], second).stdout)['projectDir'] == str(project)
del env['PI_PORTABLE_PROJECT']
override = second / 'override.json'
override.write_text(json.dumps(edited), encoding='utf-8')
env['PI_PORTABLE_CONFIG'] = str(override)
assert json.loads(run(['--portable-diagnostics'], second).stdout)['configFile'] == str(override)
del env['PI_PORTABLE_CONFIG']
inspect('check')
fixtures = json.loads((root / 'resume-fixtures.json').read_text(encoding='utf-8'))
print('Reopening all three saved conversations through the bundled CLI...', flush=True)
for session in fixtures:
    result = run(['--session', session['path'], '--mode', 'rpc'], second,
        records=json.dumps({'id': 'resume-check', 'type': 'get_messages'}) + '\n')
    records = [json.loads(line) for line in result.stdout.splitlines() if line.strip()]
    response = next(record for record in records if record.get('id') == 'resume-check')
    assert response['success'] and f"FIXTURE_{session['name']}" in json.dumps(response['data'])
    assert not result.stderr.strip()
report = {'platform': args.platform, 'automaticProjectFolder': True, 'stableAcrossLaunchDirectories': True,
    'projectConfigPreserved': True, 'configAndProjectOverrides': True, 'nativeResumeEntries': 3,
    'nativeSessionHistoryRestored': True, 'legacyOriginalsPreserved': True,
    'fixtureBasedNoModelCalls': True}
(release / f'acceptance-project-{args.platform}.json').write_text(json.dumps(report, indent=2) + '\n', encoding='utf-8')
print(json.dumps(report, indent=2))
