'use client'

import React, { ChangeEvent, useTransition } from 'react';
import { useLocale } from 'next-intl';
import { useRouter } from 'next/navigation';

export default function LocalSwitcher(props) {

  const [isPending, startTransition] = useTransition()
  const router = useRouter()
  const localActive = useLocale();

  const onSelectChange = (e: ChangeEvent<HTMLSelectElement>) => {
    const nextLocale = e.target.value;
    startTransition(() => {
      router.replace(`${nextLocale}`)
    })
  }

  return (
      <label className="border-2 rounded">
        <p className="sr-only">change language</p>
        <select
            defaultValue={localActive}
            className='bg-transparent py-2'
            onChange={onSelectChange}
            disabled={isPending}
        >
          <option value="ru">Русский</option>
          <option value="en">Английский</option>
          <option value="de">Немецкий</option>
        </select>
      </label>
  );
}
