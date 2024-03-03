import { useTranslations } from 'next-intl';

export default function Footer(props) {
  const t = useTranslations('Footer')
  return (
      <div className='my-10 text-center'>{t('title')}</div>
  );
}
