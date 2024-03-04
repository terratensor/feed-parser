import { ReactNode } from 'react';
import {Inter} from 'next/font/google';
import { locales } from '@/config';
import { getTranslations, unstable_setRequestLocale } from 'next-intl/server';
import clsx from 'clsx';
import Navigation from '@/components/Navigation';


const inter = Inter({subsets: ['latin']});

type Props = {
  children: ReactNode;
  params: {locale: string};
};

export function generateStaticParams() {
  return locales.map((locale) => ({locale}));
}

export async function generateMetadata({
 params: {locale}
}: Omit<Props, 'children'>) {
  const t = await getTranslations({locale, namespace: 'LocaleLayout'});

  return {
    title: t('title')
  };
}

export default async function LocaleLayout({
 children,
 params: {locale}
}: Props) {
  // Enable static rendering
  unstable_setRequestLocale(locale);

  return (
      <html className="h-full" lang={locale}>
      <body className={clsx(inter.className, 'flex h-full flex-col')}>
      <Navigation />
      {children}
      </body>
      </html>
  );
}
