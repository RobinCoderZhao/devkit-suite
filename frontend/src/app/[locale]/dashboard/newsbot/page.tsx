"use client";

import { useEffect, useState } from "react";
import { ExternalLink, Newspaper, Calendar, HelpCircle } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useLocale, useTranslations } from "next-intl";

interface NewsItem {
    id: number;
    title: string;
    source: string;
    url: string;
    summary: string;
    published_at: string;
}

export default function NewsbotPage() {
    const t = useTranslations("NewsBot");
    const locale = useLocale();
    const [feed, setFeed] = useState<NewsItem[]>([]);
    const [loading, setLoading] = useState(true);
    const [submitting, setSubmitting] = useState(false);
    const [subMessage, setSubMessage] = useState("");
    const [isSubscribed, setIsSubscribed] = useState(false);
    const [subLangs, setSubLangs] = useState("");
    const [subscriptions, setSubscriptions] = useState<any[]>([]);
    const [targetType, setTargetType] = useState("email");
    const [targetID, setTargetID] = useState("");
    const [targetLang, setTargetLang] = useState("auto");

    useEffect(() => {
        fetch(`/api/newsbot/feed?lang=${locale}`, {
            credentials: "include",
        })
            .then((res) => {
                if (!res.ok) throw new Error("Failed to load news feed");
                return res.json();
            })
            .then((json) => {
                setFeed(json.feed || []);
                setIsSubscribed(json.is_subscribed || false);
                setSubLangs(json.subscription_langs || "");
                setSubscriptions(json.subscriptions || []);
                setLoading(false);
            })
            .catch((err) => {
                console.error(err);
                setLoading(false);
            });
    }, []);

    if (loading) {
        return <div className="text-center p-12 text-muted-foreground animate-pulse">{t("loading")}</div>;
    }

    const handleSubscribe = async (e: React.FormEvent) => {
        e.preventDefault();
        setSubmitting(true);
        setSubMessage("");

        try {
            const res = await fetch("/api/newsbot/subscribe", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({
                    target_type: targetType,
                    target_id: targetID,
                    languages: targetLang === "auto" ? "" : targetLang
                }),
            });

            if (!res.ok) throw new Error("Subscription failed");
            setSubMessage(t("success_sub"));
            setTargetID("");
            setIsSubscribed(true);
            setSubLangs(targetLang === "auto" ? "auto" : targetLang);

            // Reload subscriptions to update the list
            fetch(`/api/newsbot/feed?lang=${locale}`, { credentials: "include" })
                .then(r => r.json())
                .then(j => setSubscriptions(j.subscriptions || []));
        } catch (err: any) {
            setSubMessage(t("failed_sub"));
        } finally {
            setSubmitting(false);
        }
    };

    const groupedFeed = feed.reduce((acc, item) => {
        const d = new Date(item.published_at);
        const dateStr = d.toLocaleDateString(locale, { weekday: "long", month: "long", day: "numeric", year: "numeric" });
        if (!acc[dateStr]) acc[dateStr] = [];
        acc[dateStr].push(item);
        return acc;
    }, {} as Record<string, NewsItem[]>);

    return (
        <div className="space-y-8 max-w-5xl mx-auto pb-12">
            <div>
                <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
                    {t("title")}
                    <TooltipProvider delayDuration={100}>
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <HelpCircle className="w-5 h-5 text-muted-foreground hover:text-primary transition-colors cursor-help" />
                            </TooltipTrigger>
                            <TooltipContent side="right" className="max-w-xs text-sm">
                                <p>{t("tooltip")}</p>
                            </TooltipContent>
                        </Tooltip>
                    </TooltipProvider>
                </h1>
                <p className="text-muted-foreground mt-2">
                    {t("subtitle")}
                </p>
            </div>

            <Card className="bg-primary/5 border-primary/20">
                <CardHeader>
                    <CardTitle className="text-xl flex items-center gap-2">
                        <Newspaper className="w-5 h-5 text-primary" />
                        {t("subscribe_title")}
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="flex flex-col gap-8">
                        {subscriptions.length > 0 && (
                            <div className="space-y-3">
                                <div className="flex items-center justify-between">
                                    <h3 className="text-sm font-medium text-muted-foreground uppercase">{t("active_subscriptions", { fallback: "Active Subscriptions" })}</h3>
                                </div>
                                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                                    {subscriptions.map((sub: any) => (
                                        <div key={sub.id} className="flex flex-col p-3 border rounded-lg bg-background/50 hover:border-primary/50 transition-colors">
                                            <div className="flex justify-between items-start">
                                                <span className="font-medium text-sm text-foreground">{sub.target_type.toUpperCase()}</span>
                                                <span className="text-xs px-2 py-0.5 bg-green-500/10 rounded text-green-600 dark:text-green-400">Active</span>
                                            </div>
                                            <span className="text-sm text-muted-foreground mt-1 truncate" title={sub.target_id}>{sub.target_id}</span>
                                            <div className="text-xs text-muted-foreground mt-2">Lang: {sub.languages || "auto"}</div>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        )}

                        <div className="space-y-4">
                            <h3 className="text-sm font-medium text-muted-foreground uppercase">{t("add_subscription", { fallback: "Add New Subscription" })}</h3>
                            <form onSubmit={handleSubscribe} className="flex flex-col gap-6 max-w-xl">
                                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                                    <div className="space-y-2">
                                        <Label className="text-muted-foreground text-xs font-semibold uppercase">{t("channel")}</Label>
                                        <Select value={targetType} onValueChange={setTargetType}>
                                            <SelectTrigger className="bg-background">
                                                <SelectValue />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="email">{t("channel_email")}</SelectItem>
                                                <SelectItem value="feishu">{t("channel_feishu")}</SelectItem>
                                                <SelectItem value="slack">{t("channel_slack")}</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </div>
                                    <div className="space-y-2">
                                        <Label className="text-muted-foreground text-xs font-semibold uppercase">{t("language")}</Label>
                                        <Select value={targetLang} onValueChange={setTargetLang}>
                                            <SelectTrigger className="bg-background">
                                                <SelectValue />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="auto">{t("lang_auto")}</SelectItem>
                                                <SelectItem value="zh">{t("lang_zh")}</SelectItem>
                                                <SelectItem value="en">{t("lang_en")}</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </div>
                                </div>

                                <div className="space-y-2">
                                    <Label className="text-sm font-semibold">{targetType === "email" ? t("channel_email") : "Webhook URL"}</Label>
                                    <div className="flex flex-col sm:flex-row gap-3 mt-1">
                                        <Input
                                            type={targetType === "email" ? "email" : "url"}
                                            placeholder={targetType === "email" ? t("subscribe_placeholder") : t("webhook_placeholder")}
                                            value={targetID}
                                            onChange={(e) => setTargetID(e.target.value)}
                                            className="bg-background flex-1 min-w-0"
                                            required
                                        />
                                        <Button type="submit" disabled={submitting} className="shrink-0">
                                            {submitting ? t("subscribing_btn") : t("subscribe_btn")}
                                        </Button>
                                    </div>
                                </div>
                            </form>
                        </div>
                    </div>

                    {subMessage && (
                        <p className="mt-3 text-sm font-medium text-muted-foreground">{subMessage}</p>
                    )}
                </CardContent>
            </Card>

            <div className="space-y-12 mt-10">
                {Object.entries(groupedFeed).map(([date, items]) => (
                    <div key={date} className="space-y-6 relative">
                        <div className="flex items-center gap-4">
                            <h2 className="text-xl font-bold text-foreground flex items-center gap-2 shrink-0">
                                <Calendar className="w-5 h-5 text-primary" />
                                {date}
                            </h2>
                            <div className="h-px bg-muted flex-1"></div>
                        </div>

                        <div className="grid gap-6">
                            {items.map((item) => (
                                <Card key={item.id} className="transition-all hover:border-primary/50 hover:shadow-md">
                                    <CardHeader className="pb-3">
                                        <div className="flex items-center justify-between mb-2">
                                            <div className="flex items-center gap-2 text-xs font-semibold text-muted-foreground bg-muted px-2 py-1 rounded-md">
                                                <Newspaper className="w-3 h-3" />
                                                {item.source}
                                            </div>
                                            <div className="flex items-center gap-1 text-xs text-muted-foreground">
                                                {new Date(item.published_at).toLocaleTimeString(locale, { hour: '2-digit', minute: '2-digit' })}
                                            </div>
                                        </div>
                                        <CardTitle className="text-xl leading-tight">
                                            {item.title}
                                        </CardTitle>
                                    </CardHeader>
                                    <CardContent>
                                        <p className="text-muted-foreground leading-relaxed mb-4">
                                            {item.summary}
                                        </p>
                                        <Button variant="outline" size="sm" asChild>
                                            <a href={item.url} target="_blank" rel="noreferrer" className="flex items-center gap-2">
                                                {t("read_original")} <ExternalLink className="w-3 h-3" />
                                            </a>
                                        </Button>
                                    </CardContent>
                                </Card>
                            ))}
                        </div>
                    </div>
                ))}

                {feed.length === 0 && (
                    <div className="text-center py-12 text-muted-foreground border-dashed border-2 rounded-xl">
                        {t("no_signals")}
                    </div>
                )}
            </div>
        </div>
    );
}
