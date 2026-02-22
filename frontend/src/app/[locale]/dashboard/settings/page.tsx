"use client";

import { useEffect, useState } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { AlertCircle, CreditCard, BellRing, Settings2, LayoutDashboard, Newspaper } from "lucide-react";
import { useTranslations } from "next-intl";

export default function SettingsPage() {
    const t = useTranslations("Settings");
    const [userPlan, setUserPlan] = useState("free");
    const [rules, setRules] = useState<any[]>([]);
    const [newRule, setNewRule] = useState({ rule_type: "severity", rule_value: "high", target_type: "email", target_id: "" });
    const [loadingRules, setLoadingRules] = useState(true);

    const [profile, setProfile] = useState({ name: "", phone: "", company: "" });
    const [savingProfile, setSavingProfile] = useState(false);
    const [newsbotSubs, setNewsbotSubs] = useState<any[]>([]);
    const [loadingSubs, setLoadingSubs] = useState(true);

    useEffect(() => {
        // Fetch User Details for Billing and Profile
        fetch("/api/users/me", {
            method: "GET",
            headers: { "Content-Type": "application/json" },
            credentials: "include"
        }).then(res => res.json()).then(data => {
            if (data.plan) setUserPlan(data.plan);
            if (data.email) {
                setProfile({
                    name: data.name || "",
                    phone: data.phone || "",
                    company: data.company || ""
                });
            }
        }).catch(() => { });

        // Fetch Alert Rules and NewsBot Subscriptions
        fetchAlertRules();
        fetchNewsbotSubs();
    }, []);

    const fetchAlertRules = async () => {
        try {
            const res = await fetch("/api/watchbot/rules", {
                credentials: "include"
            });
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

    const fetchNewsbotSubs = async () => {
        try {
            const res = await fetch("/api/newsbot/subscriptions", { credentials: "include" });
            if (res.ok) {
                const data = await res.json();
                setNewsbotSubs(data.subscriptions || []);
            }
        } catch (e) {
            console.error(e);
        } finally {
            setLoadingSubs(false);
        }
    };

    const handleUpdateProfile = async () => {
        setSavingProfile(true);
        try {
            const res = await fetch("/api/users/profile", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify(profile)
            });
            if (res.ok) {
                alert(t("profile.saved"));
            }
        } catch (e) {
            console.error(e);
        } finally {
            setSavingProfile(false);
        }
    };

    const handleDeleteNewsbotSub = async (id: number) => {
        try {
            const res = await fetch(`/api/newsbot/subscriptions?id=${id}`, {
                method: "DELETE",
                credentials: "include"
            });
            if (res.ok) {
                fetchNewsbotSubs();
            }
        } catch (e) {
            console.error(e);
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
                    target_type: newRule.target_type,
                    target_id: newRule.target_id
                })
            });
            if (res.ok) {
                setNewRule({ rule_type: "severity", rule_value: "high", target_type: "email", target_id: "" });
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
                    <TabsTrigger value="profile" className="gap-2 data-[state=active]:bg-background">
                        <Settings2 className="w-4 h-4" /> {t("tabs.profile")}
                    </TabsTrigger>
                    <TabsTrigger value="alerts" className="gap-2 data-[state=active]:bg-background">
                        <LayoutDashboard className="w-4 h-4" /> {t("tabs.alerts")}
                    </TabsTrigger>
                    <TabsTrigger value="newsbot" className="gap-2 data-[state=active]:bg-background">
                        <Newspaper className="w-4 h-4" /> {t("tabs.newsbot")}
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
                                        value={newRule.target_type} onChange={e => setNewRule({ ...newRule, target_type: e.target.value })}
                                    >
                                        <option value="email">{t("alerts.form.act_email")}</option>
                                        <option value="feishu">Feishu</option>
                                        <option value="slack">Slack</option>
                                    </select>
                                </div>
                                <div className="space-y-2 col-span-full">
                                    <Label>{newRule.target_type === "email" ? "Email Address" : "Webhook URL"}</Label>
                                    <Input
                                        type={newRule.target_type === "email" ? "email" : "url"}
                                        placeholder={newRule.target_type === "email" ? "e.g. alerts@example.com" : "https://hooks..."}
                                        value={newRule.target_id} onChange={e => setNewRule({ ...newRule, target_id: e.target.value })}
                                        className="bg-background"
                                        required
                                    />
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
                                                        <p className="text-xs text-muted-foreground">{t("alerts.rule_then")} {rule.target_type.toUpperCase()} ({rule.target_id})</p>
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

                <TabsContent value="profile" className="space-y-6">
                    <Card>
                        <CardHeader>
                            <CardTitle>{t("profile.title")}</CardTitle>
                            <CardDescription>{t("profile.desc")}</CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            <div className="space-y-2">
                                <Label>{t("profile.name_label")}</Label>
                                <Input value={profile.name} onChange={(e) => setProfile({ ...profile, name: e.target.value })} />
                            </div>
                            <div className="space-y-2">
                                <Label>{t("profile.phone_label")}</Label>
                                <Input value={profile.phone} onChange={(e) => setProfile({ ...profile, phone: e.target.value })} />
                            </div>
                            <div className="space-y-2">
                                <Label>{t("profile.company_label")}</Label>
                                <Input value={profile.company} onChange={(e) => setProfile({ ...profile, company: e.target.value })} />
                            </div>
                            <Button onClick={handleUpdateProfile} disabled={savingProfile}>
                                {savingProfile ? t("profile.saving") : t("profile.save")}
                            </Button>
                        </CardContent>
                    </Card>
                </TabsContent>

                <TabsContent value="newsbot" className="space-y-6">
                    <Card>
                        <CardHeader>
                            <CardTitle>{t("newsbot.title")}</CardTitle>
                            <CardDescription>{t("newsbot.desc")}</CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            <h3 className="text-lg font-medium flex items-center gap-2"><Settings2 className="w-4 h-4" /> {t("newsbot.active")}</h3>
                            {loadingSubs ? (
                                <p className="text-sm text-muted-foreground">{t("loading_rules")}</p>
                            ) : newsbotSubs.length === 0 ? (
                                <p className="text-sm text-muted-foreground italic">{t("newsbot.no_subs")}</p>
                            ) : (
                                <div className="space-y-3">
                                    {newsbotSubs.map((sub: any) => (
                                        <div key={sub.id} className="flex justify-between items-center p-3 border rounded-lg hover:border-primary/50 transition-colors">
                                            <div className="flex items-center gap-3">
                                                <div>
                                                    <p className="text-sm font-medium">
                                                        <span className="text-primary">{sub.target_type.toUpperCase()}</span>: <span className="font-semibold">{sub.target_id}</span>
                                                    </p>
                                                    <p className="text-xs text-muted-foreground">{t("newsbot.lang")}: {sub.languages}</p>
                                                </div>
                                            </div>
                                            <Button onClick={() => handleDeleteNewsbotSub(sub.id)} variant="ghost" size="sm" className="text-red-500 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-950">
                                                {t("newsbot.remove")}
                                            </Button>
                                        </div>
                                    ))}
                                </div>
                            )}
                        </CardContent>
                    </Card>
                </TabsContent>
            </Tabs>
        </div>
    );
}
