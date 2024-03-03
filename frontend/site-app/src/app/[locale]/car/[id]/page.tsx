'use client'

import React from 'react';
import { NextPage } from 'next';

import { usePathname, useSearchParams } from 'next/navigation'
import { router } from 'next/client';

const CarPage: NextPage = () => {

  const pathname = usePathname()
  const searchParams = useSearchParams()

  console.log(pathname, searchParams)

  return (
      <div>
        Car page
      </div>
  );
};

export default CarPage;
