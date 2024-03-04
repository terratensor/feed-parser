import React, { Suspense } from 'react';
import { fetchData } from '../../../lib/data';

export default async function DataDisplay({ id, include }: { id: string | undefined, include: string | undefined }) {

  const data = await fetchData(id, include)

  return (
      <Suspense fallback={<div>loading…</div>}>
      </Suspense>
  );
}
