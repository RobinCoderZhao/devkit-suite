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
import { useTranslations } from "next-intl";

export default function SettingsPage() {
    const t = useTranslations("Settings");
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
                alert(t("alerts.failed_create"));
            }
        } catch (e) {
            console.error(e);
        }
    };

    return (
        <div className="max-w-5xl mx-auto space-y-8">
            <div>
                <h2 className="text-3xl font-bold tracking-tight">{t("title")}</h2>
                <p className="text-muted-foreground mt-2">{t("subtitle")}</p>
            </div>

            <Tabs defaultValue="billing" className="space-y-6">
                <TabsList className="bg-muted/50 w-full justify-start h-12 p-1 border">
                    <TabsTrigger value="billing" className="gap-2 data-[state=active]:bg-background">
                        <CreditCard className="w-4 h-4" /> {t("tabs.billing")}
                    </TabsTrigger>
                    <TabsTrigger value="alerts" className="gap-2 data-[state=active]:bg-background">
                        <BellRing className="w-4 h-4" /> {t("tabs.alerts")}
                    </TabsTrigger>
                </TabsList>

                <TabsContent value="billing" className="space-y-6">
                    <Card>
                        <CardHeader>
                            <CardTitle>{t("billing.title")}</CardTitle>
                            <CardDescription>{t("billing.current", { plan: userPlan.toUpperCase() })}</CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            <div className="flex items-center justify-between p-4 border rounded-lg bg-muted/20">
                                <div className="space-y-1">
                                    <p className="font-medium text-lg flex items-center gap-2">
                                        {userPlan === "pro" ? t("billing.pro") : t("billing.free")}
                                        {userPlan === "pro" && <Badge className="bg-primary/20 text-primary border-primary/30">{t("billing.active")}</Badge>}
                                    </p>
                                    <p className="text-sm text-muted-foreground">
                                        {userPlan === "pro"
                                            ? t("billing.pro_desc")
                                            : t("billing.free_desc")}
                                    </p>
                                </div>
                                {userPlan === "free" && (
                                    <Button asChild>
                                        <a href="/pricing">{t("billing.upgrade")}</a>
                                    </Button>
                                )}
                                {userPlan === "pro" && (
                                    <Button variant="outline">
                                        {t("billing.manage")}
                                    </Button>
                                )}
                            </div>
                        </CardContent>
                    </Card>
                </TabsContent>

                <TabsContent value="alerts" className="space-y-6">
                    <Card>
                        <CardHeader>
                            <CardTitle>{t("alerts.title")}</CardTitle>
                            <CardDescription>{t("alerts.desc")}</CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-6">

                            {userPlan === "free" && (
                                <div className="bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20 p-4 rounded-lg flex items-start gap-3">
                                    <AlertCircle className="w-5 h-5 shrink-0 mt-0.5" />
                                    <div>
                                        <p className="font-semibold">{t("alerts.pro_feature")}</p>
                                        <p className="text-sm mt-1">{t("alerts.pro_feature_desc")}</p>
                                    </div>
                                </div>
                            )}

                            <div className="grid gap-4 md:grid-cols-4 items-end border p-4 rounded-lg bg-muted/10 relative">
                                {userPlan === "free" && <div className="absolute inset-0 bg-background/50 backdrop-blur-[1px] z-10 rounded-lg pointer-events-none" />}

                                <div className="space-y-2">
                                    <Label>{t("alerts.form.cond_type")}</Label>
                                    <select
                                        className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                                        value={newRule.rule_type} onChange={e => setNewRule({ ...newRule, rule_type: e.target.value })}
                                    >
                                        <option value="severity">{t("alerts.form.val_severity")}</option>
                                        <option value="keyword">{t("alerts.form.val_keyword")}</option>
                                    </select>
                                </div>
                                <div className="space-y-2">
                                    <Label>{t("alerts.form.target")}</Label>
                                    {newRule.rule_type === "severity" ? (
                                        <select
                                            className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                                            value={newRule.rule_value} onChange={e => setNewRule({ ...newRule, rule_value: e.target.value })}
                                        >
                                            <option value="high">{t("alerts.form.high")}</option>
                                            <option value="medium">{t("alerts.form.medium")}</option>
                                            <option value="low">{t("alerts.form.low")}</option>
                                        </select>
                                    ) : (
                                        <Input
                                            placeholder="e.g. 'pricing', 'enterprise'"
                                            value={newRule.rule_value} onChange={e => setNewRule({ ...newRule, rule_value: e.target.value })}
                                        />
                                    )}
                                </div>
                                <div className="space-y-2">
                                    <Label>{t("alerts.form.action")}</Label>
                                    <select
                                        className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                                        value={newRule.action} onChange={e => setNewRule({ ...newRule, action: e.target.value })}
                                    >
                                        <option value="email">{t("alerts.form.act_email")}</option>
                                        <option value="webhook">{t("alerts.form.act_webhook")}</option>
                                    </select>
                                </div>
                                <Button onClick={handleCreateRule} disabled={userPlan === "free"} className="w-full h-9">
                                    {t("alerts.form.add")}
                                </Button>
                            </div>

                            <div className="space-y-4 pt-4 border-t">
                                <h3 className="text-lg font-medium flex items-center gap-2"><Settings2 className="w-4 h-4" /> {t("alerts.active")}</h3>
                                {loadingRules ? (
                                    <p className="text-sm text-muted-foreground">{t("loading_rules")}</p>
                                ) : rules.length === 0 ? (
                                    <p className="text-sm text-muted-foreground italic">{t("alerts.no_rules")}</p>
                                ) : (
                                    <div className="space-y-3">
                                        {rules.map((rule) => (
                                            <div key={rule.id} className="flex justify-between items-center p-3 border rounded-lg hover:border-primary/50 transition-colors">
                                                <div className="flex items-center gap-3">
                                                    <Switch checked={rule.is_active} disabled={userPlan === "free"} />
                                                    <div>
                                                        <p className="text-sm font-medium">
                                                            {t("alerts.rule_if")} <span className="text-primary">{rule.rule_type}</span> {t("alerts.rule_eq")} <span className="font-semibold">{rule.rule_value}</span>
                                                        </p>
                                                        <p className="text-xs text-muted-foreground">{t("alerts.rule_then")} {rule.action.toUpperCase()}</p>
                                                    </div>
                                                </div>
                                                <Button variant="ghost" size="sm" className="text-red-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-950">{t("alerts.remove")}</Button>
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
