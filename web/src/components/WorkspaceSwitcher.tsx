import { Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'
import type { Workspace } from '../lib/types'
import { useTranslation } from '../lib/i18n'

export function WorkspaceSwitcher({ workspaces, activeWorkspaceId, defaultWorkspaceId, onChange, onCreate, onDelete }: { workspaces: Workspace[]; activeWorkspaceId?: string; defaultWorkspaceId?: string; onChange: (workspaceId: string) => void; onCreate: () => void; onDelete?: (workspaceId: string) => Promise<void> }) {
  const { t } = useTranslation()
  const shellLabels = t('shell')
  const [confirmingId, setConfirmingId] = useState<string>()
  const [deleting, setDeleting] = useState(false)
  const [error, setError] = useState<string>()

  const displayName = (workspace: Workspace) => (workspace.display_name === 'Chưa có tên' || workspace.display_name === 'Unnamed Workspace')
    ? shellLabels.unnamed_workspace
    : workspace.display_name

  const isDefault = activeWorkspaceId && defaultWorkspaceId && activeWorkspaceId === defaultWorkspaceId
  const canDelete = onDelete && workspaces.length > 1 && activeWorkspaceId && !isDefault

  const confirmDelete = async () => {
    if (!onDelete || !confirmingId) return
    setError(undefined)
    setDeleting(true)
    try {
      await onDelete(confirmingId)
      setConfirmingId(undefined)
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : shellLabels.delete_business_error)
    } finally {
      setDeleting(false)
    }
  }

  return <div className="workspace-switcher">
    <label htmlFor="workspace-select">{shellLabels.selected_business}</label>
    <div className="workspace-select-row">
      <select id="workspace-select" value={activeWorkspaceId ?? ''} onChange={event => onChange(event.target.value)} aria-label={shellLabels.selected_business}>
        {workspaces.map(workspace => <option key={workspace.id} value={workspace.id}>{displayName(workspace)}</option>)}
      </select>
      <button type="button" className="workspace-add" aria-label={shellLabels.add_business} title={shellLabels.add_business} onClick={onCreate}><Plus aria-hidden="true" /></button>
      {onDelete && (
        <button
          type="button"
          className="workspace-delete"
          aria-label={shellLabels.delete_business}
          title={canDelete ? shellLabels.delete_business : shellLabels.delete_business_default_hint}
          disabled={!canDelete}
          onClick={() => activeWorkspaceId && setConfirmingId(activeWorkspaceId)}
        >
          <Trash2 aria-hidden="true" />
        </button>
      )}
    </div>
    {confirmingId && (
      <div className="workspace-delete-confirm" role="alertdialog" aria-label={shellLabels.delete_business_confirm_title}>
        <strong>{shellLabels.delete_business_confirm_title}</strong>
        <p>{shellLabels.delete_business_confirm_desc}</p>
        {error && <p className="inline-error" role="alert">{error}</p>}
        <div className="workspace-delete-actions">
          <button type="button" className="button secondary" disabled={deleting} onClick={() => { setConfirmingId(undefined); setError(undefined) }}>{shellLabels.delete_business_cancel_btn}</button>
          <button type="button" className="button danger" disabled={deleting} onClick={() => void confirmDelete()}>{shellLabels.delete_business_confirm_btn}</button>
        </div>
      </div>
    )}
  </div>
}
