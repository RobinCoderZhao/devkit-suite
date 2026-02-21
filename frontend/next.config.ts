import _createNextIntlPlugin from 'next-intl/plugin';
import type { NextConfig } from "next";

const withNextIntl = _createNextIntlPlugin(
  './src/i18n/request.ts'
);

const nextConfig: NextConfig = {
  /* config options here */
};

export default withNextIntl(nextConfig);
