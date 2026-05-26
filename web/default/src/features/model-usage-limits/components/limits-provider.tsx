import { createContext, useContext, useState, type ReactNode } from 'react'

type LimitsState = {
  // Mutate drawer
  open: 'create' | 'update' | null
  setOpen: (open: 'create' | 'update' | null) => void
  currentRow: Record<string, unknown> | null
  setCurrentRow: (row: Record<string, unknown> | null) => void
  // Delete dialog
  deleteOpen: boolean
  setDeleteOpen: (open: boolean) => void
  deleteRow: Record<string, unknown> | null
  setDeleteRow: (row: Record<string, unknown> | null) => void
  // Refresh
  refreshTrigger: number
  triggerRefresh: () => void
}

const LimitsContext = createContext<LimitsState | null>(null)

export function LimitsProvider({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState<'create' | 'update' | null>(null)
  const [currentRow, setCurrentRow] = useState<Record<string, unknown> | null>(null)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deleteRow, setDeleteRow] = useState<Record<string, unknown> | null>(null)
  const [refreshTrigger, setRefreshTrigger] = useState(0)

  const triggerRefresh = () => setRefreshTrigger((n) => n + 1)

  return (
    <LimitsContext.Provider
      value={{
        open, setOpen,
        currentRow, setCurrentRow,
        deleteOpen, setDeleteOpen,
        deleteRow, setDeleteRow,
        refreshTrigger, triggerRefresh,
      }}
    >
      {children}
    </LimitsContext.Provider>
  )
}

export function useLimits() {
  const ctx = useContext(LimitsContext)
  if (!ctx) throw new Error('useLimits must be used within LimitsProvider')
  return ctx
}
