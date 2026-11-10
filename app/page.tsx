// Root route (/) page implementing a responsive 70/30 desktop layout (Tailwind lg and above)
// with a 70% creative hero illustration section and a 30% tabbed authentication section (Login/Register).
import Image from "next/image";
import AuthTabs from "@/feature/auth/AuthTabs";

export default function Home() {
  return (
    <div className="mx-auto flex min-h-[calc(100dvh-8rem)] w-full max-w-7xl flex-1 flex-col justify-center overflow-x-hidden px-4 py-8 sm:px-6 lg:flex-row lg:items-center lg:gap-8 lg:px-8 lg:py-12">
      {/* 70% Information Section: displays the NutriSync hero illustration on desktop (lg+);
          hidden below lg to avoid unnecessary vertical scroll and prioritize authentication on smaller screens. */}
      <section
        aria-label="Information"
        className="hidden lg:block lg:w-[70%] lg:shrink-0"
      >
        <div className="relative aspect-1376/768 w-full overflow-hidden rounded-2xl border shadow-sm">
          <Image
            src="/nutrisync-hero.jpg"
            alt="NutriSync AI health companion illustration combining human vitality, botanical nutrition, and biometric telemetry waves"
            width={1376}
            height={768}
            sizes="(min-width: 1024px) 70vw, 0px"
            fetchPriority="high"
            loading="eager"
            className="h-full w-full object-cover"
          />
        </div>
      </section>

      {/* 30% Authentication Section: prioritized on mobile/tablet and occupies 30% width on desktop (lg+). */}
      <section
        aria-label="Authentication"
        className="flex w-full justify-center lg:w-[30%] lg:shrink-0"
      >
        <AuthTabs />
      </section>
    </div>
  );
}
