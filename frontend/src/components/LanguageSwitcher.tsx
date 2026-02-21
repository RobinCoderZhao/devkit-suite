"use client";

import { useLocale } from "next-intl";
import { usePathname, useRouter } from "@/i18n/routing";
import { Button } from "@/components/ui/button";
import { Globe } from "lucide-react";

export function LanguageSwitcher() {
    const locale = useLocale();
    const router = useRouter();
    const pathname = usePathname();

    const toggleLanguage = () => {
        const nextLocale = locale === "en" ? "zh" : "en";
        router.replace(pathname, { locale: nextLocale });
    };

    return (
        <Button variant="ghost" size="sm" onClick={toggleLanguage} className="gap-2 text-muted-foreground hover:text-foreground">
            <Globe className="w-4 h-4" />
            {locale === "en" ? "中文" : "English"}
        </Button>
    );
}
