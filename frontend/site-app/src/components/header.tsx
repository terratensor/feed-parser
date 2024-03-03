import React from 'react';
import Link from 'next/link';
import { useTranslations } from 'next-intl';
import LocalSwitcher from '@/components/local-switcher';

function Header(props) {
  const t = useTranslations('Navigation')

  return (
      <header className='p-4'>
        <nav className='flex items-center justify-between'>
          <Link href='/'>{t('home')}</Link>
          <LocalSwitcher />
        </nav>
      </header>
  );
}

export default Header;
