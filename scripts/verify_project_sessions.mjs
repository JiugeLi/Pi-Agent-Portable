// Exercise the pinned native catalogue used by /resume. Fixtures never call AI.
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

const [mode, infoFile, launchDir] = process.argv.slice(2);
const info = JSON.parse(fs.readFileSync(infoFile, 'utf8'));
process.env.PI_CODING_AGENT_DIR = info.agentDir;
process.env.PI_PACKAGE_DIR = path.join(info.runtime, 'pi');
const { SessionManager } = await import(pathToFileURL(path.join(info.runtime, 'pi/dist/core/session-manager.js')).href);
const metadata = path.join(path.dirname(infoFile), 'project-fixtures.json');

if (mode === 'create') {
  const created = [];
  for (const name of ['project-one', 'project-two', 'legacy-one']) {
    const legacy = name === 'legacy-one';
    const manager = SessionManager.create(legacy ? launchDir : info.cwd,
      legacy ? path.join(launchDir, '.pi-portable/agent/sessions/old-folder') : undefined);
    manager.appendMessage({role: 'user', content: `FIXTURE_${name}`, timestamp: Date.now()});
    manager.appendSessionInfo(name);
    created.push({name, path: manager.getSessionFile(), cwd: manager.getCwd(),
      original: fs.readFileSync(manager.getSessionFile(), 'utf8')});
  }
  fs.writeFileSync(metadata, JSON.stringify(created));
  console.log('Created three native session fixtures; no model calls.');
} else {
  const sessions = await SessionManager.list(info.cwd);
  if (mode === 'real') {
    assert(sessions.length > 0, 'No real session was persisted');
    assert(sessions.some(session => JSON.stringify(SessionManager.open(session.path).buildSessionContext().messages)
      .includes('PORTABLE_TEST_OK')), 'Real conversation was not restored');
    console.log(JSON.stringify({realSessionSaved: true, nativeResumeCatalogue: true, nativeHistoryRestored: true}));
  } else {
    const expected = JSON.parse(fs.readFileSync(metadata, 'utf8'));
    assert.equal(sessions.length, expected.length, 'Missing or duplicated resume entries');
    const restored = expected.map(fixture => {
      const session = sessions.find(item => item.name === fixture.name);
      assert(session, 'Session absent from the initial /resume catalogue');
      const manager = SessionManager.open(session.path);
      assert.equal(manager.getCwd(), fixture.cwd, 'Migration changed session working-directory metadata');
      assert(JSON.stringify(manager.buildSessionContext().messages).includes(`FIXTURE_${fixture.name}`));
      assert.equal(fs.readFileSync(fixture.path, 'utf8'), fixture.original, 'Original session changed');
      return {path: session.path, name: fixture.name};
    });
    fs.writeFileSync(path.join(path.dirname(infoFile), 'resume-fixtures.json'), JSON.stringify(restored));
    console.log(JSON.stringify({nativeResumeEntries: sessions.length, nativeHistoryRestored: true,
      legacySourcesPreserved: true, legacyCwdPreserved: true}));
  }
}
