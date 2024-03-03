import React, { FC } from 'react';
import Link from 'next/link';
import styles from './Header.module.css'

const Header: FC = () => {
  return (
      <div className={styles.header}>
        <Link href={'/'}>Home</Link>
        <Link href={'/about'}>About page</Link>
      </div>
  );
};

export default Header;
