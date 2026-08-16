import { Link, useLocation } from "react-router-dom";
import { Video } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ROUTES } from "@/lib/constants";
import { cn } from "@/lib/utils";

type SidebarNavProps = {
  isCollapsed?: boolean;
};

export default function SidebarNav({ isCollapsed }: SidebarNavProps) {
  const location = useLocation();
  const isActive = (path: string) => location.pathname === path;

  return (
    <div className="px-3">
      <div className="space-y-1">
        <Link to={ROUTES.interviewIntro}>
          <Button
            variant={isActive(ROUTES.interviewIntro) ? "secondary" : "ghost"}
            className={cn(
              "w-full rounded-full",
              isCollapsed ? "justify-center" : "justify-start",
            )}
          >
            <Video className={cn("h-4 w-4", !isCollapsed && "mr-2")} />
            {!isCollapsed && "AI 面试"}
          </Button>
        </Link>
      </div>
    </div>
  );
}
