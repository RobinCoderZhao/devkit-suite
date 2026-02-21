import { getRequestConfig } from 'next-intl/server';
import { notFound } from 'next/navigation';
import fs from 'fs/promises';
import path from 'path';

const locales = ['en', 'zh'];

export default getRequestConfig(async ({ requestLocale }) => {
    let locale = await requestLocale;
    console.log("getRequestConfig invoked. requestLocale extracted:", locale);

    if (!locale || !locales.includes(locale as any)) {
        console.error("request.ts rejected locale:", locale);
        locale = 'en';
    }

    const filePath = path.join(process.cwd(), 'messages', `${locale}.json`);
    const fileContent = await fs.readFile(filePath, 'utf8');
    const messages = JSON.parse(fileContent);

    return {
        locale,
        messages
    } as any;
});

