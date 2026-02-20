"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowRight, AlertTriangle, ShieldCheck, Activity, Globe } from "lucide-react";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

interface DashboardCompetitor {
    id: number;
    name: string;
    domain: string;
    pages_tracked: number;
    latest_change_time: string | null;
    latest_severity: string;
    recent_alert_snippet: string;
}

interface DashboardData {
    competitors: DashboardCompetitor[];
}

export default function DashboardPage() {
    const [data, setData] = useState<DashboardData | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        fetch("http://localhost:8080/api/watchbot/dashboard", {
            credentials: "include", // send JWT cookie
        })
            .then((res) => res.json())
            .then((json) => {
                setData(json);
                setLoading(false);
            })
            .catch((err) => {
                console.error(err);
                setLoading(false);
            });
    }, []);

    if (loading) {
        return <div className="text-center p-12 text-muted-foreground animate-pulse">Loading Battlecards...</div>;
    }

    const competitors = data?.competitors || [];

    return (
        <div className="space-y-8">
            <div>
                <h1 className="text-3xl font-bold tracking-tight">Battlecards</h1>
                <p className="text-muted-foreground mt-2">
                    Your global overview of tracked competitors.
                </p>
            </div>

            {competitors.length === 0 ? (
                <Card className="flex flex-col items-center justify-center min-h-[400px] border-dashed">
                    <CardHeader className="text-center">
                        <CardTitle>No Competitors Watched</CardTitle>
                        <CardDescription>Add a domain to start tracking their public shifts immediately.</CardDescription>
                    </CardHeader>
                    <CardContent>
                        <Button asChild>
                            <Link href="/onboarding">Run Onboarding Again</Link>
                        </Button>
                    </CardContent>
                </Card>
            ) : (
                <div className="grid gap-6 md:grid-cols-2 xl:grid-cols-3">
                    {competitors.map((comp) => (
                        <Card key={comp.id} className="flex flex-col transition-all hover:border-primary/50 hover:shadow-md">
                            <CardHeader className="pb-4">
                                <div className="flex items-start justify-between">
                                    <div>
                                        <CardTitle className="text-xl">{comp.name}</CardTitle>
                                        <CardDescription className="flex items-center gap-1 mt-1">
                                            <Globe className="w-3 h-3" />
                                            {comp.domain}
                                        </CardDescription>
                                    </div>
                                    <div className={`px-2.5 py-1 text-xs font-semibold rounded-full border ${comp.latest_severity === 'critical' ? 'bg-destructive/10 text-destructive border-destructive/20' :
                                            comp.latest_severity === 'important' ? 'bg-orange-500/10 text-orange-500 border-orange-500/20' :
                                                'bg-muted text-muted-foreground'
                                        }`}>
                                        {comp.latest_severity ? comp.latest_severity.toUpperCase() : "NO CHANGES YET"}
                                    </div>
                                </div>
                            </CardHeader>
                            <CardContent className="flex-1">
                                <div className="grid grid-cols-2 gap-4 mb-4">
                                    <div className="space-y-1">
                                        <p className="text-xs font-medium text-muted-foreground">Pages Tracked</p>
                                        <p className="text-2xl font-bold">{comp.pages_tracked}</p>
                                    </div>
                                    <div className="space-y-1">
                                        <p className="text-xs font-medium text-muted-foreground">Activity</p>
                                        <div className="flex items-center gap-1 text-sm font-medium text-primary">
                                            <Activity className="w-4 h-4" />
                                            Active
                                        </div>
                                    </div>
                                </div>

                                <div className="p-3 bg-muted/50 rounded-lg text-sm">
                                    {comp.latest_change_time ? (
                                        <div className="space-y-2">
                                            <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
                                                <AlertTriangle className="w-3.5 h-3.5" />
                                                Latest Shift
                                            </div>
                                            <p className="line-clamp-2 text-foreground/80 leading-relaxed font-mono">
                                                {comp.recent_alert_snippet || "Structural change detected..."}
                                            </p>
                                        </div>
                                    ) : (
                                        <div className="flex items-center gap-2 text-muted-foreground">
                                            <ShieldCheck className="w-4 h-4" />
                                            Currently baseline monitoring.
                                        </div>
                                    )}
                                </div>
                            </CardContent>
                            <CardFooter className="pt-4 border-t bg-muted/10">
                                <Button variant="ghost" className="w-full justify-between" asChild>
                                    <Link href={`/dashboard/watchbot/${comp.id}`}>
                                        View Timeline & Diffs
                                        <ArrowRight className="w-4 h-4" />
                                    </Link>
                                </Button>
                            </CardFooter>
                        </Card>
                    ))}
                </div>
            )}
        </div>
    );
}
