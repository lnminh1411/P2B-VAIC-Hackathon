import { Plus } from 'lucide-react'
import type { Workspace } from '../lib/types'
import { useTranslation } from '../lib/i18n'

export function WorkspaceSwitcher({ workspaces, activeWorkspaceId, onChange, onCreate }: { workspaces: Workspace[]; activeWorkspaceId?: string; onChange: (workspaceId: string) => void; onCreate: () => void }) {
  const { t } = useTranslation()
  const shellLabels = t('shell')
  
  return <div className="workspace-switcher">
    <label htmlFor="workspace-select">{shellLabels.selected_business}</label>
    <div className="workspace-select-row">
      <select id="workspace-select" value={activeWorkspaceId ?? ''} onChange={event => onChange(event.target.value)} aria-label={shellLabels.selected_business}>
        {workspaces.map(workspace => {
          const name = (workspace.display_name === 'Chưa có tên' || workspace.display_name === 'Unnamed Workspace')
            ? shellLabels.unnamed_workspace
            : workspace.display_name
          return <option key={workspace.id} value={workspace.id}>{name}</option>
        })}
      </select>
      <button type="button" className="workspace-add" aria-label={shellLabels.add_business} onClick={onCreate}><Plus aria-hidden="true" /></button>
    </div>
  </div>
}
