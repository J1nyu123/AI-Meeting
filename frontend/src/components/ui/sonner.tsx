import { Toaster as Sonner, type ToasterProps } from "sonner";

export function Toaster(props: ToasterProps) {
  return (
    <Sonner
      position="top-center"
      richColors
      closeButton
      toastOptions={{
        duration: 3500,
        classNames: {
          toast: "font-sans",
        },
      }}
      {...props}
    />
  );
}
