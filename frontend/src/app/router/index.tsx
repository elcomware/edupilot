import { createBrowserRouter } from 'react-router-dom'
import { AppShell } from '../shell/AppShell'
import { DashboardPage } from './DashboardPage'
import { PeopleListPage } from '@/platform/people/PeopleListPage'
import { PersonDetailPage } from '@/platform/people/PersonDetailPage'
import { PlaceholderPage } from './PlaceholderPage'
import { NAVIGATION } from '../shell/navigation'

/**
 * Navigation entries that have a real screen behind them. Every other entry
 * falls through to PlaceholderPage. The list is explicit so that a section link
 * and its implemented page cannot both claim the same path and leave the
 * router picking whichever happened to be declared first.
 */
const IMPLEMENTED_PATHS: ReadonlySet<string> = new Set(['people'])

/**
 * One layout route owns the shell; every navigation entry becomes a leaf
 * route. Feature modules add their own routes as they are implemented, and the
 * dashboard is the first page backed by real data.
 */
export const router = createBrowserRouter([
  {
    path: '/',
    element: <AppShell />,
    children: [
      { index: true, element: <DashboardPage /> },
      // A person screen is reached from the directory, not from the navigation,
      // so it is declared here rather than added to NAVIGATION.
      { path: 'people/:id', element: <PersonDetailPage /> },
      { path: 'people', element: <PeopleListPage /> },
      ...NAVIGATION.flatMap((section) =>
        section.items
          .filter((item) => !IMPLEMENTED_PATHS.has(item.path.slice(1)))
          .map((item) => ({
            path: item.path.slice(1),
            element: <PlaceholderPage titleKey={item.key} />,
          })),
      ),
    ],
  },
])
