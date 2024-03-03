import React, { FC, PropsWithChildren } from 'react';
import Header from '@/components/layout/header/Header';

const Layout: FC<PropsWithChildren<unknown>> = ({ children }) => {
  return (<div>
    <Header />
    {children}
  </div>);
};

export default Layout;
