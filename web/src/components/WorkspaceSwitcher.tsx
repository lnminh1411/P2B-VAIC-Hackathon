import { Plus } from 'lucide-react'
import type { Workspace } from '../lib/types'

export function WorkspaceSwitcher({ workspaces, activeWorkspaceId, onChange, onCreate }: { workspaces: Workspace[]; activeWorkspaceId?: string; onChange: (workspaceId: string) => void; onCreate: () => void }) {
  return <div className="workspace-switcher">
    <label htmlFor="workspace-select">Doanh nghiệp đang chọn</label>
    <div className="workspace-select-row">
      <select id="workspace-select" value={activeWorkspaceId ?? ''} onChange={event => onChange(event.target.value)} aria-label="Doanh nghiệp đang chọn">
        {workspaces.map(workspace => <option key={workspace.id} value={workspace.id}>{workspace.display_name}</option>)}
      </select>
      <button type="button" className="workspace-add" aria-label="Thêm doanh nghiệp" onClick={onCreate}><Plus aria-hidden="true" /></button>
    </div>
  </div>
}
