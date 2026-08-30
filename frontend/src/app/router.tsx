import { createBrowserRouter } from "react-router-dom";
import { RequireAuth } from "@/features/auth/RequireAuth";
import { InterviewWorkspaceLayout } from "@/layouts/InterviewWorkspaceLayout";
import { RootLayout } from "@/layouts/RootLayout";
import { AuthPage } from "@/pages/AuthPage";
import { HomePage } from "@/pages/HomePage";
import { InterviewRoomPage } from "@/pages/InterviewRoomPage";
import { InterviewReportPage } from "@/pages/InterviewReportPage";
import { InterviewsPage } from "@/pages/InterviewsPage";
import { NewInterviewPage } from "@/pages/NewInterviewPage";

export const router = createBrowserRouter([
  {
    element: <RootLayout />,
    children: [
      {
        path: "/",
        element: <HomePage />,
      },
      {
        path: "/auth",
        element: <AuthPage />,
      },
      {
        element: <RequireAuth />,
        children: [
          {
            element: <InterviewWorkspaceLayout />,
            children: [
              {
                path: "/interviews",
                element: <InterviewsPage />,
              },
              {
                path: "/interviews/new",
                element: <NewInterviewPage />,
              },
              {
                path: "/interviews/:sessionId",
                element: <InterviewRoomPage />,
              },
              {
                path: "/interviews/:sessionId/report",
                element: <InterviewReportPage />,
              },
            ],
          },
        ],
      },
    ],
  },
]);
