import { test } from 'node:test';
import assert from 'node:assert/strict';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import portableUI from '../runtime/extensions/portable-ui.ts';

const upstream = process.env.PI_PACKAGE_DIR ?? path.join(process.env.APPDATA, 'npm/node_modules/@earendil-works/pi-coding-agent');
process.env.PI_PACKAGE_DIR = upstream;
const { visibleWidth } = await import(pathToFileURL(path.join(upstream, 'node_modules/@earendil-works/pi-tui/dist/index.js')).href);

async function fixture(mode = 'tui') {
  const events = new Map(), commands = new Map();
  let footer, expanded = false;
  const pi = { on: (name, handler) => events.set(name, handler),
    registerCommand: (name, command) => commands.set(name, command), getThinkingLevel: () => 'medium' };
  const ctx = { mode, cwd: 'D:\\工作\\a long project folder\\子目录'.repeat(3),
    model: { id: 'a-long-model-name-with-wide-中文-text'.repeat(3) },
    getContextUsage: () => ({percent: 24.6}),
    ui: { setTitle() {}, setFooter: factory => { footer = factory; },
      notify() {}, getToolsExpanded: () => expanded, setToolsExpanded: value => { expanded = value; } } };
  await portableUI(pi);
  events.get('session_start')({}, ctx);
  return { ctx, commands, footer, expanded: () => expanded };
}

test('footer fits narrow terminals, ANSI colors and wide characters', async () => {
  const { footer } = await fixture();
  let disposed = false;
  const component = footer({requestRender() {}}, { fg: (_color, text) => `\x1b[36m${text}\x1b[0m` }, {
    onBranchChange: () => () => { disposed = true; }, getGitBranch: () => '分支-feature',
    getExtensionStatuses: () => new Map([['test', 'wide-状态'.repeat(50)]]) });
  for (const width of [1, 20, 40, 60, 80, 120]) {
    const lines = component.render(width);
    assert.equal(lines.length, 4);
    assert(lines.every(line => visibleWidth(line) <= width), `footer exceeded ${width} columns`);
  }
  assert(component.render(120).some(line => line.includes('Ctrl+O')));
  component.dispose();
  assert(disposed);
});

test('/tools toggles native folding, with explicit show/hide and invalid input handling', async () => {
  const { ctx, commands, expanded } = await fixture();
  const tools = commands.get('tools');
  await tools.handler('', ctx); assert(expanded());
  await tools.handler('hide', ctx); assert(!expanded());
  await tools.handler('show', ctx); assert(expanded());
  await tools.handler('unknown', ctx); assert(expanded());
});

test('print, JSON and RPC modes install no terminal UI', async () => {
  for (const mode of ['print', 'json', 'rpc']) {
    const { ctx, commands, footer, expanded } = await fixture(mode);
    assert.equal(footer, undefined);
    await commands.get('tools').handler('', ctx);
    assert(!expanded());
  }
});
