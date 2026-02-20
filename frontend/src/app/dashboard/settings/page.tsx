"use client";

import { useEffect, useState } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { AlertCircle, CreditCard, BellRing, Settings2 } from "lucide-react";

export default function SettingsPage() {
    const [userPlan, setUserPlan] = useState("free");
    const [rules, setRules] = useState<any[]>([]);
    const [newRule, setNewRule] = useState({ rule_type: "severity", rule_value: "high", action: "email" });
    const [loadingRules, setLoadingRules] = useState(true);

    useEffect(() => {
        // Fetch User Details for Billing
        fetch("/api/auth/login", {
            method: "POST",
            body: JSON.stringify({ email: "test@example.com", password: "mock" }),
            headers: { "Content-Type": "application/json" }
        }).then(res => res.json()).then(data => {
            if (data.plan) setUserPlan(data.plan);
        }).catch(() => { });

        // Fetch Alert Rules
        fetchAlertRules();
    }, []);

    const fetchAlertRules = async () => {
        try {
            const res = await fetch("/api/watchbot/rules");
            if (res.ok) {
                const data = await res.json();
                setRules(data.rules || []);
            }
        } catch (e) {
            console.error(e);
        } finally {
            setLoadingRules(false);
        }
    };

    const handleCreateRule = async () => {
        try {
            const res = await fetch("/api/watchbot/rules", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    rule_type: newRule.rule_type,
                    rule_value: newRule.rule_value,
                    action: newRule.action
                })
            });
            if (res.ok) {
                setNewRule({ rule_type: "severity", rule_value: "high", action: "email" });
                fetchAlertRules();
            } else {
                alert("Failed to create rule");
            }
        } catch (e) {
            console.error(e);
        }
    };

    return (
        <div className="max-w-5xl mx-auto space-y-8">
            <div>
                <h2 className="text-3xl font-bold tracking-tight">Settings</h2>
                <p className="text-muted-foreground mt-2">Manage your account settings, billing, and alert preferences.</p>
            </div>

            <Tabs defaultValue="billing" className="space-y-6">
                <TabsList className="bg-muted/50 w-full justify-start h-12 p-1 border">
                    <TabsTrigger value="billing" className="gap-2 data-[state=active]:bg-background">
                        <CreditCard className="w-4 h-4" /> Billing & Plan
                    </TabsTrigger>
                    <TabsTrigger value="alerts" className="gap-2 data-[state=active]:bg-background">
                        <BellRing className="w-4 h-4" /> Smart Alerts
                    </TabsTrigger>
                </TabsList>

                <TabsContent value="billing" className="space-y-6">
                    <Card>
                        <CardHeader>
                            <CardTitle>Subscription Plan</CardTitle>
                            <CardDescription>You are currently on the {userPlan.toUpperCase()} plan.</CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            <div className="flex items-center justify-between p-4 border rounded-lg bg-muted/20">
                                <div className="space-y-1">
                                    <p className="font-medium text-lg flex items-center gap-2">
                                        {userPlan === "pro" ? "Pro Plan" : "Free Plan"}
                                        {userPlan === "pro" && <Badge className="bg-primary/20 text-primary border-primary/30">Active</Badge>}
                                    </p>
                                    <p className="text-sm text-muted-foreground">
                                        {userPlan === "pro"
                                            ? "You have full access to real-time Smart Alerts and up to 100 competitors."
                                            : "Upgrade to Pro to unlock Smart Alerts and track more competitors."}
                                    </p>
                                </div>
                                {userPlan === "free" && (
                                    <Button asChild>
                                        <a href="/pricing">Upgrade Plan</a>
                                    </Button>
                                )}
                                {userPlan === "pro" && (
                                    <Button variant="outline">
                                        Manage Billing
                                    </Button>
                                )}
                            </div>
                        </CardContent>
                    </Card>
                </TabsContent>

                <TabsContent value="alerts" className="space-y-6">
                    <Card>
                        <CardHeader>
                            <CardTitle>Configure Smart Alerts</CardTitle>
                            <CardDescription>Define criteria for when WatchBot should send you notifications.</CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-6">

                            {userPlan === "free" && (
                                <div className="bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20 p-4 rounded-lg flex items-start gap-3">
                                    <AlertCircle className="w-5 h-5 shrink-0 mt-0.5" />
                                    <div>
                                        <p className="font-semibold">Pro Feature</p>
                                        <p className="text-sm mt-1">Smart Alerts are only available on the Pro plan. Upgrade to receive real-time notifications for critical changes.</p>
                                    </div>
                                </div>
                            )}

                            <div className="grid gap-4 md:grid-cols-4 items-end border p-4 rounded-lg bg-muted/10 relative">
                                {userPlan === "free" && <div className="absolute inset-0 bg-background/50 backdrop-blur-[1px] z-10 rounded-lg pointer-events-none" />}

                                <div className="space-y-2">
                                    <Label>Condition Type</Label>
                                    <select
                                        className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                                        value={newRule.rule_type} onChange={e => setNewRule({ ...newRule, rule_type: e.target.value })}
                                    >
                                        <option value="severity">Severity Level</option>
                                        <option value="keyword">Contains Keyword</option>
                                    </select>
                                </div>
                                <div className="space-y-2">
                                    <Label>Target Value</Label>
                                    {newRule.rule_type === "severity" ? (
                                        <select
                                            className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                                            value={newRule.rule_value} onChange={e => setNewRule({ ...newRule, rule_value: e.target.value })}
                                        >
                                            <option value="high">High & Above</option>
                                            <option value="medium">Medium & Above</option>
                                            <option value="low">All Changes</option>
                                        </select>
                                    ) : (
                                        <Input
                                            placeholder="e.g. 'pricing', 'enterprise'"
                                            value={newRule.rule_value} onChange={e => setNewRule({ ...newRule, rule_value: e.target.value })}
                                        />
                                    )}
                                </div>
                                <div className="space-y-2">
                                    <Label>Action</Label>
                                    <select
                                        className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                                        value={newRule.action} onChange={e => setNewRule({ ...newRule, action: e.target.value })}
                                    >
                                        <option value="email">Send Email</option>
                                        <option value="webhook">Trigger Webhook</option>
                                    </select>
                                </div>
                                <Button onClick={handleCreateRule} disabled={userPlan === "free"} className="w-full h-9">
                                    Add Rule
                                </Button>
                            </div>

                            <div className="space-y-4 pt-4 border-t">
                                <h3 className="text-lg font-medium flex items-center gap-2"><Settings2 className="w-4 h-4" /> Active Rules</h3>
                                {loadingRules ? (
                                    <p className="text-sm text-muted-foreground">Loading rules...</p>
                                ) : rules.length === 0 ? (
                                    <p className="text-sm text-muted-foreground italic">No alert rules configured yet.</p>
                                ) : (
                                    <div className="space-y-3">
                                        {rules.map((rule) => (
                                            <div key={rule.id} className="flex justify-between items-center p-3 border rounded-lg hover:border-primary/50 transition-colors">
                                                <div className="flex items-center gap-3">
                                                    <Switch checked={rule.is_active} disabled={userPlan === "free"} />
                                                    <div>
                                                        <p className="text-sm font-medium">
                                                            If <span className="text-primary">{rule.rule_type}</span> equals <span className="font-semibold">{rule.rule_value}</span>
                                                        </p>
                                                        <p className="text-xs text-muted-foreground">Then DO: {rule.action.toUpperCase()}</p>
                                                    </div>
                                                </div>
                                                <Button variant="ghost" size="sm" className="text-red-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-950">Remove</Button>
                                            </div>
                                        ))}
                                    </div>
                                )}
                            </div>
                        </CardContent>
                    </Card>
                </TabsContent>
            </Tabs>
        </div>
    );
}
