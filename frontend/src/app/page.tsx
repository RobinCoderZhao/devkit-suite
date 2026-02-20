import Link from "next/link";
import { Button } from "@/components/ui/button";
import { ArrowRight, Eye, Zap, Shield, Users } from "lucide-react";

export default function LandingPage() {
  return (
    <div className="flex flex-col min-h-screen">
      <header className="px-6 h-16 flex items-center border-b border-border/40 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 z-50 sticky top-0">
        <div className="flex items-center gap-2 font-bold text-xl tracking-tight">
          <Eye className="w-6 h-6 text-primary" />
          DevKit Suite
        </div>
        <nav className="ml-auto flex items-center gap-4 sm:gap-6">
          <Link href="#features" className="text-sm font-medium hover:text-primary transition-colors">
            Features
          </Link>
          <Link href="/login" className="text-sm font-medium hover:text-primary transition-colors">
            Sign In
          </Link>
          <Button asChild size="sm">
            <Link href="/register">Get Started</Link>
          </Button>
        </nav>
      </header>
      <main className="flex-1">
        <section className="w-full py-24 md:py-32 lg:py-48 flex items-center justify-center relative overflow-hidden">
          {/* Subtle gradient background */}
          <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[800px] h-[500px] bg-primary/20 rounded-full blur-[120px] opacity-50 pointer-events-none" />

          <div className="container px-4 md:px-6 relative z-10">
            <div className="flex flex-col items-center space-y-8 text-center">
              <div className="space-y-4 max-w-[800px]">
                <h1 className="text-4xl font-extrabold tracking-tight sm:text-5xl md:text-6xl lg:text-7xl">
                  Your SaaS Intelligence <br className="hidden sm:inline" />
                  <span className="text-transparent bg-clip-text bg-gradient-to-r from-primary to-blue-500">
                    Command Center
                  </span>
                </h1>
                <p className="mx-auto max-w-[700px] text-muted-foreground md:text-xl leading-relaxed">
                  Monitor competitors' pricing, features, and docs automatically.
                  Digest global AI news. Never miss an external shift again.
                </p>
              </div>
              <div className="flex flex-col sm:flex-row gap-4">
                <Button size="lg" className="h-12 px-8 text-base font-semibold" asChild>
                  <Link href="/register">
                    Start Monitoring Free
                    <ArrowRight className="ml-2 w-5 h-5" />
                  </Link>
                </Button>
                <Button size="lg" variant="outline" className="h-12 px-8 text-base font-semibold" asChild>
                  <Link href="#demo">Watch Demo</Link>
                </Button>
              </div>
            </div>
          </div>
        </section>

        <section id="features" className="w-full py-20 bg-muted/50">
          <div className="container px-4 md:px-6">
            <div className="grid gap-12 lg:grid-cols-3">
              <div className="space-y-4">
                <div className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
                  <Zap className="w-6 h-6" />
                </div>
                <h3 className="text-xl font-bold">WatchBot AI Diffing</h3>
                <p className="text-muted-foreground">
                  Stop manually checking competitor pricing pages. Our LLM pipeline detects meaningful changes and summarizes them instantly.
                </p>
              </div>
              <div className="space-y-4">
                <div className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
                  <Shield className="w-6 h-6" />
                </div>
                <h3 className="text-xl font-bold">Smart Alerts</h3>
                <p className="text-muted-foreground">
                  Get notified only when it matters. Configure rules to alert your team via email or Slack when high-severity changes occur.
                </p>
              </div>
              <div className="space-y-4">
                <div className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
                  <Users className="w-6 h-6" />
                </div>
                <h3 className="text-xl font-bold">Team Workspaces</h3>
                <p className="text-muted-foreground">
                  Share insights, comment on external changes, and coordinate your go-to-market response within a unified team dashboard.
                </p>
              </div>
            </div>
          </div>
        </section>
      </main>
      <footer className="w-full flex-col sm:flex-row py-6 shrink-0 items-center justify-center px-4 md:px-6 border-t flex">
        <p className="text-sm text-muted-foreground">
          © {new Date().getFullYear()} DevKit Suite. All rights reserved.
        </p>
      </footer>
    </div>
  );
}
