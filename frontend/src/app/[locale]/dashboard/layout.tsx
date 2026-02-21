import { Link } from "@/i18n/routing";
import { Eye, Settings, LayoutDashboard, Newspaper, LogOut } from "lucide-react";
import { LanguageSwitcher } from "@/components/LanguageSwitcher";

export default function DashboardLayout({
    children,
}: {
    children: React.ReactNode;
}) {
    return (
        <div className="flex min-h-screen">
            {/* Sidebar */}
            <aside className="w-64 border-r bg-muted/30 p-4 flex flex-col gap-6 sticky top-0 h-screen">
                <Link href="/dashboard" className="flex items-center gap-2 font-bold text-xl px-2">
                    <Eye className="w-6 h-6 text-primary" />
                    DevKit Suite
                </Link>
                <nav className="flex flex-col gap-2 flex-1">
                    <Link href="/dashboard" className="flex items-center gap-2 px-2 py-2 text-sm font-medium rounded-md hover:bg-muted/60 transition-colors">
                        <LayoutDashboard className="w-4 h-4" />
                        WatchBot
                    </Link>
                    <Link href="/dashboard/newsbot" className="flex items-center gap-2 px-2 py-2 text-sm font-medium rounded-md hover:bg-muted/60 transition-colors">
                        <Newspaper className="w-4 h-4" />
                        NewsBot
                    </Link>
                    <Link href="/dashboard/settings" className="flex items-center gap-2 px-2 py-2 text-sm font-medium rounded-md hover:bg-muted/60 transition-colors mt-auto">
                        <Settings className="w-4 h-4" />
                        Settings
                    </Link>
                </nav>
                <div className="border-t pt-4 flex flex-col gap-2">
                    <LanguageSwitcher />
                    <button className="flex items-center gap-2 px-2 py-2 text-sm font-medium rounded-md hover:bg-muted/60 transition-colors w-full text-left text-muted-foreground hover:text-foreground">
                        <LogOut className="w-4 h-4" />
                        Log Out
                    </button>
                </div>
            </aside>

            {/* Main Content */}
            <main className="flex-1 overflow-auto bg-background p-8">
                {children}
            </main>
        </div>
    );
}
