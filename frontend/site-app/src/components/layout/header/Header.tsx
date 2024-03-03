'use client';

import React, { FC } from 'react';
import Link from 'next/link';
import styles from './Header.module.scss'
import { usePathname } from 'next/navigation';

const Header: FC = () => {

  const pathname = usePathname()

  return (
      <header className={styles.header}>
        <Link href={'/'} className={pathname === '/' ? styles.active : ''}>Home</Link>
        <Link href={'/about'} className={pathname==='/about' ? styles.active : ''}>About page</Link>
      </header>
  );
};

export default Header;
