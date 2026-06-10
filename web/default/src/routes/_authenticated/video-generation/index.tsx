import { createFileRoute, redirect } from '@tanstack/react-router'
import { Main } from '@/components/layout'
import { VideoGenerationPlayground } from '@/features/video-generation'
import { isSidebarModuleEnabled } from '@/lib/nav-modules'

export const Route = createFileRoute('/_authenticated/video-generation/')({
  beforeLoad: () => {
    if (!isSidebarModuleEnabled('chat', 'video_generation')) {
      throw redirect({ to: '/dashboard' })
    }
  },
  component: VideoGenerationPage,
})

function VideoGenerationPage() {
  return (
    <Main className='p-0'>
      <VideoGenerationPlayground />
    </Main>
  )
}