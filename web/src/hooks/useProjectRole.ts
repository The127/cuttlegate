import { useQuery } from '@tanstack/react-query'
import { fetchJSON } from '../api'
import { getUserManager } from '../auth'

type ProjectRole = 'admin' | 'editor' | 'viewer'

interface Member {
  user_id: string
  role: string
}

export function useProjectRole(projectSlug: string) {
  return useQuery({
    queryKey: ['project-role', projectSlug],
    queryFn: async (): Promise<ProjectRole | null> => {
      const [membersData, user] = await Promise.all([
        fetchJSON<{ members: Member[] }>(`/api/v1/projects/${projectSlug}/members`),
        getUserManager().getUser(),
      ])
      const sub = user?.profile.sub
      if (!sub) return null
      const me = membersData.members.find((m) => m.user_id === sub)
      return (me?.role as ProjectRole) ?? null
    },
  })
}
