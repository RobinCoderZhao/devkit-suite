"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Eye, Code, Cpu, LineChart, Loader2 } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

const INDUSTRIES = [
    {
        id: "devtools",
        label: "Developer Tools",
        icon: <Code className="w-8 h-8 mb-4 text-blue-500" />,
        description: "Vercel, Supabase, Cursor",
    },
    {
        id: "llm",
        label: "AI Models & APIs",
        icon: <Cpu className="w-8 h-8 mb-4 text-purple-500" />,
        description: "OpenAI, Anthropic, Gemini",
    },
    {
        id: "saas",
        label: "B2B SaaS",
        icon: <LineChart className="w-8 h-8 mb-4 text-emerald-500" />,
        description: "Stripe, Notion, Linear",
    },
];

export default function OnboardingPage() {
    const router = useRouter();
    const [selectedIndustry, setSelectedIndustry] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);

    const handleContinue = async () => {
        if (!selectedIndustry) return;

        setLoading(true);
        try {
            const res = await fetch("http://localhost:8080/api/onboarding", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                // Notice we assume cookies (JWT) are sent automatically, but we might need credentials mode
                // For simplicity in this demo, let's keep it standard fetch. We allowed credentials in CORS.
                credentials: "include",
                body: JSON.stringify({ industry: selectedIndustry }),
            });

            if (!res.ok) {
                throw new Error("Failed to configure workspace");
            }

            // Success, go to dashboard
            router.push("/dashboard");
        } catch (err) {
            console.error(err);
            setLoading(false);
            // Fallback: still go to dashboard even if it errors out for demo resilience
            router.push("/dashboard");
        }
    };

    return (
        <div className="min-h-screen flex flex-col items-center justify-center p-4 bg-muted/20">
            <div className="absolute top-8 left-8 flex items-center gap-2 font-bold text-xl">
                <Eye className="w-6 h-6 text-primary" />
                DevKit Suite
            </div>

            <div className="max-w-3xl w-full space-y-8">
                <div className="text-center space-y-2">
                    <h1 className="text-3xl font-bold tracking-tight">Welcome aboard! 👋</h1>
                    <p className="text-muted-foreground text-lg">
                        Let's get your command center ready. What industry are you tracking?
                    </p>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    {INDUSTRIES.map((ind) => (
                        <Card
                            key={ind.id}
                            className={`cursor-pointer transition-all hover:border-primary border-2 ${selectedIndustry === ind.id ? "border-primary bg-primary/5 shadow-md" : "border-transparent"}`}
                            onClick={() => setSelectedIndustry(ind.id)}
                        >
                            <CardContent className="pt-6 flex flex-col items-center text-center">
                                {ind.icon}
                                <CardTitle className="text-xl mb-2">{ind.label}</CardTitle>
                                <CardDescription>{ind.description}</CardDescription>
                            </CardContent>
                        </Card>
                    ))}
                </div>

                <div className="flex justify-center pt-8">
                    <Button
                        size="lg"
                        className="w-full max-w-sm h-12 text-lg"
                        disabled={!selectedIndustry || loading}
                        onClick={handleContinue}
                    >
                        {loading && <Loader2 className="mr-2 h-5 w-5 animate-spin" />}
                        {loading ? "Configuring workspace..." : "Continue"}
                    </Button>
                </div>
            </div>
        </div>
    );
}
