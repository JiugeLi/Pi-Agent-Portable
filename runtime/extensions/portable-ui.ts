import type { ExtensionAPI } from '@earendil-works/pi-coding-agent';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

export default async function (pi: ExtensionAPI) {
  const { truncateToWidth } = await import(pathToFileURL(path.join(
    process.env.PI_PACKAGE_DIR!, 'node_modules/@earendil-works/pi-tui/dist/index.js')).href);

  pi.registerCommand('tools', {
    description: 'Show or hide full tool output (also Ctrl+O)',
    handler: async (args, ctx) => {
      if (ctx.mode !== 'tui') return;
      const action = args.trim().toLowerCase();
      if (action && !['show', 'hide'].includes(action)) {
        ctx.ui.notify('Use /tools, /tools show, or /tools hide. Ctrl+O also toggles output.', 'info');
        return;
      }
      ctx.ui.setToolsExpanded(action ? action === 'show' : !ctx.ui.getToolsExpanded());
    },
  });

  pi.on('session_start', (_event, ctx) => {
    if (ctx.mode !== 'tui') return;
    ctx.ui.setTitle('Pi Agent Portable');
    ctx.ui.setFooter((tui, theme, data) => {
      const unsubscribe = data.onBranchChange(() => tui.requestRender());
      return {
        dispose: unsubscribe,
        invalidate() {},
        render(width: number) {
          const fit = (text: string) => truncateToWidth(text, Math.max(0, width));
          const branch = data.getGitBranch();
          const context = ctx.getContextUsage();
          const usage = context?.percent == null ? '' : ` | context ${context.percent.toFixed(0)}%`;
          const model = ctx.model?.id ?? 'select a model with /model';
          const hints = width >= 90
            ? 'Ctrl+O tool output | Ctrl+T thinking | /model | /resume | /settings | Esc stop'
            : width >= 55 ? 'Ctrl+O tools | Ctrl+T thinking | /model | Esc stop'
            : 'Ctrl+O tools | /model | Esc stop';
          const statuses = [...data.getExtensionStatuses().values()].join(' | ');
          return [
            fit(theme.fg('muted', `${ctx.cwd}${branch ? ` (${branch})` : ''}`)),
            fit(theme.fg('accent', model) + theme.fg('muted', ` | ${pi.getThinkingLevel()}${usage}`)),
            fit(theme.fg('dim', hints)),
            ...(statuses ? [fit(statuses)] : []),
          ];
        },
      };
    });
  });
}
