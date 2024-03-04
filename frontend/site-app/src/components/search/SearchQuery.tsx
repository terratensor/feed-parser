'use client'

import React from 'react';
import { useSearchParams } from 'next/navigation';

export default function SearchQuery(props) {
  const searchParams = useSearchParams()
  const q = searchParams.get("q")
  const lang = searchParams.get("lang")

  const data = {q:q || "",lang:lang || ""}
  return <div>{JSON.stringify(data, undefined, 2)}</div>;
}
