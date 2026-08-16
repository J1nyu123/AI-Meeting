import AuthMarketingPanel from "@/components/auth/AuthMarketingPanel";
import AuthFormCard from "@/components/auth/AuthFormCard";
import { useAuthPageController } from "@/hooks/auth/useAuthPageController";
import { AUTH_BACKGROUND_VIDEO_SRC } from "@/pages/auth/authPage.constants";

export default function AuthPage() {
  const {
    mode,
    formData,
    loading,
    error,
    registerLoading,
    localError,
    switchMode,
    handleInputChange,
    handleSubmit,
  } = useAuthPageController();

  return (
    <div className="relative flex h-full w-full items-center justify-center overflow-hidden bg-gradient-to-br from-slate-950 via-blue-950 to-cyan-900 px-6 py-10">
      <div className="pointer-events-none absolute inset-0">
        <video
          aria-hidden="true"
          className="h-full w-full object-cover motion-reduce:hidden"
          src={AUTH_BACKGROUND_VIDEO_SRC}
          autoPlay
          muted
          loop
          playsInline
          preload="metadata"
        />
        <div className="absolute inset-0 bg-slate-950/65" />
        <div className="absolute inset-0 bg-[linear-gradient(115deg,rgba(15,23,42,0.25),rgba(14,116,144,0.15),rgba(37,99,235,0.20))]" />
      </div>

      <div className="relative w-full max-w-5xl grid gap-10 lg:grid-cols-[1.1fr_0.9fr] items-center">
        <AuthMarketingPanel />
        <AuthFormCard
          mode={mode}
          formData={formData}
          errorMessage={error || localError}
          isSubmitting={loading || registerLoading}
          onSwitchMode={switchMode}
          onInputChange={handleInputChange}
          onSubmit={handleSubmit}
        />
      </div>
    </div>
  );
}
