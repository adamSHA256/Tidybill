import { useState, useEffect } from 'react'
import { Outlet } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Center, Loader } from '@mantine/core'
import { AppShell } from './components/AppShell'
import { MobileShell } from './components/MobileShell'
import { useIsMobile } from './hooks/useIsMobile'
import { ApiHealthGuard } from './components/ApiHealthGuard'
import { SetupWizard } from './pages/SetupWizard'
import { api } from './api/client'
import { TourProvider } from './tour/TourProvider'
import { WelcomeTourDialog } from './tour/WelcomeTourDialog'
import { readState } from './tour/persistence'

export default function AppLayout() {
  const [wizardDone, setWizardDone] = useState(false)
  const [welcomeOpen, setWelcomeOpen] = useState(false)

  const { data: firstRunData, isLoading, isError } = useQuery({
    queryKey: ['first-run'],
    queryFn: api.getFirstRun,
    retry: 3,
    retryDelay: 1000,
  })

  const showWizard = !wizardDone && firstRunData?.first_run === true
  const isMobile = useIsMobile()
  const Shell = isMobile ? MobileShell : AppShell

  useEffect(() => {
    if (showWizard || isLoading) return
    const state = readState()
    if (!state.welcomeSeen && !state.doNotAutoShow) {
      setWelcomeOpen(true)
    }
  }, [showWizard, isLoading])

  return (
    <ApiHealthGuard>
      {(isLoading || (isError && !firstRunData)) ? (
        <Center h="100vh">
          <Loader />
        </Center>
      ) : showWizard ? (
        <SetupWizard onComplete={() => setWizardDone(true)} />
      ) : (
        <TourProvider>
          <Shell>
            <Outlet />
          </Shell>
          <WelcomeTourDialog
            opened={welcomeOpen}
            onClose={() => setWelcomeOpen(false)}
          />
        </TourProvider>
      )}
    </ApiHealthGuard>
  )
}

export function ResponsivePage({ Desktop, Mobile }: { Desktop: React.ComponentType; Mobile: React.ComponentType }) {
  const isMobile = useIsMobile()
  return isMobile ? <Mobile /> : <Desktop />
}
