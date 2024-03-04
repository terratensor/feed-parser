import { useTranslations } from 'next-intl';
import { unstable_setRequestLocale } from 'next-intl/server';
import PageLayout from '@/components/PageLayout';

type Props = {
  params: { locale: string };
  searchParams: { q: string | undefined }
};

export default function IndexPage({ params: { locale }, searchParams: { q } }: Props) {
  // Enable static rendering
  unstable_setRequestLocale(locale);
  console.log(q)
  const t = useTranslations('IndexPage')

  return (
      <PageLayout title={t('title')}>
        <p className="max-w-[590px]">
          {t.rich('description', {
            code: (chunks) => (
                <code className="font-mono text-white">{chunks}</code>
            )
          })}
        </p>
      </PageLayout>
  );
}
