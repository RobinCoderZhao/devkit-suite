"use client";

import { Button } from "@/components/ui/button";
import { Check, ShieldAlert } from "lucide-react";
import { Link, useRouter } from "@/i18n/routing";
import { useEffect, useState } from "react";

export default function PricingPage() {
    const router = useRouter();
    const [loading, setLoading] = useState(false);
    const [plan, setPlan] = useState("free");

    useEffect(() => {
        fetch("/api/users/me").then(res => {
            if (res.ok) {
                // Here we could fetch the plan if /api/users/me exposes it
                // we added /api/auth/login that returns the plan. 
                // For simplicity, let's assume if it exists, they are logged in.
            } else {
                // router.push("/login?redirect=/pricing");
            }
        });

        const urlParams = new URLSearchParams(window.location.search);
        if (urlParams.get("checkout") === "cancel") {
            alert("Checkout canceled");
        }
    }, [router]);

    const handleCheckout = async (priceId: string) => {
        setLoading(true);
        try {
            const res = await fetch("/api/billing/create-checkout-session", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ price_id: priceId }),
            });
            if (res.ok) {
                const { url } = await res.json();
                window.location.href = url;
            } else if (res.status === 401) {
                router.push("/login?redirect=/pricing");
            } else {
                alert("Failed to create checkout session");
            }
        } catch (err) {
            console.error(err);
        }
        setLoading(false);
    };

    return (
        <div className="container py-24 mx-auto max-w-5xl">
            <div className="text-center mb-16 px-4">
                <h1 className="text-4xl font-extrabold tracking-tight sm:text-5xl md:text-6xl text-transparent bg-clip-text bg-gradient-to-r from-primary to-blue-500">
                    Simple, transparent pricing
                </h1>
                <p className="mt-4 text-xl text-muted-foreground max-w-2xl mx-auto">
                    Choose the plan that's right for your intelligence needs.
                </p>
            </div>

            <div className="grid md:grid-cols-3 gap-8 px-4">
                {/* Free Plan */}
                <div className="rounded-2xl border border-border/40 p-8 bg-card shadow-sm flex flex-col">
                    <h3 className="text-2xl font-bold mb-2">Free</h3>
                    <p className="text-muted-foreground mb-6">For individuals starting out.</p>
                    <div className="text-4xl font-extrabold mb-8">$0<span className="text-lg font-normal text-muted-foreground">/mo</span></div>

                    <ul className="space-y-4 mb-8 flex-1">
                        <li className="flex items-center"><Check className="text-primary w-5 h-5 mr-3 shrink-0" /> 2 Competitors tracked</li>
                        <li className="flex items-center"><Check className="text-primary w-5 h-5 mr-3 shrink-0" /> Daily summary sync</li>
                        <li className="flex items-center"><Check className="text-primary w-5 h-5 mr-3 shrink-0" /> Basic Diff Viewer</li>
                    </ul>

                    <Button variant="outline" className="w-full h-12" disabled>Current Plan</Button>
                </div>

                {/* Pro Plan */}
                <div className="rounded-2xl border-2 border-primary p-8 bg-card shadow-lg flex flex-col relative scale-[1.02] md:-mt-4 md:mb-4">
                    <div className="absolute top-0 right-8 -translate-y-1/2 bg-primary text-primary-foreground px-3 py-1 rounded-full text-sm font-semibold">
                        Most Popular
                    </div>
                    <h3 className="text-2xl font-bold mb-2 text-primary">Pro</h3>
                    <p className="text-muted-foreground mb-6">For growth-focused teams.</p>
                    <div className="text-4xl font-extrabold mb-8">$29<span className="text-lg font-normal text-muted-foreground">/mo</span></div>

                    <ul className="space-y-4 mb-8 flex-1">
                        <li className="flex items-center font-medium"><Check className="text-primary w-5 h-5 mr-3 shrink-0" /> 100 Competitors tracked</li>
                        <li className="flex items-center"><ShieldAlert className="text-blue-500 w-5 h-5 mr-3 shrink-0" /> Real-time Smart Alerts</li>
                        <li className="flex items-center"><Check className="text-primary w-5 h-5 mr-3 shrink-0" /> LLM Analysis & Summarization</li>
                        <li className="flex items-center"><Check className="text-primary w-5 h-5 mr-3 shrink-0" /> API Access</li>
                    </ul>

                    <Button onClick={() => handleCheckout("price_1Q_Mock_Pro")} disabled={loading} className="w-full h-12 shadow-md">
                        {loading ? "Redirecting..." : "Upgrade to Pro"}
                    </Button>
                </div>

                {/* Enterprise Plan */}
                <div className="rounded-2xl border border-border/40 p-8 bg-card shadow-sm flex flex-col">
                    <h3 className="text-2xl font-bold mb-2">Enterprise</h3>
                    <p className="text-muted-foreground mb-6">For large organizations.</p>
                    <div className="text-4xl font-extrabold mb-8">Custom<span className="text-lg font-normal text-muted-foreground"></span></div>

                    <ul className="space-y-4 mb-8 flex-1">
                        <li className="flex items-center"><Check className="text-primary w-5 h-5 mr-3 shrink-0" /> Unlimited tracking</li>
                        <li className="flex items-center"><Check className="text-primary w-5 h-5 mr-3 shrink-0" /> Custom data sources (JoySpace)</li>
                        <li className="flex items-center"><Check className="text-primary w-5 h-5 mr-3 shrink-0" /> SAML SSO</li>
                        <li className="flex items-center"><Check className="text-primary w-5 h-5 mr-3 shrink-0" /> Dedicated account manager</li>
                    </ul>

                    <Button asChild variant="secondary" className="w-full h-12">
                        <Link href="mailto:sales@devkit-suite.com">Contact Sales</Link>
                    </Button>
                </div>
            </div>
        </div>
    );
}
