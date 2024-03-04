import {notFound} from 'next/navigation';
import {getRequestConfig} from 'next-intl/server';
import {locales} from './config';

export default getRequestConfig(async ({locale}) => {
  // Validate that the incoming `locale` parameter is valid
  if (!locales.includes(locale as any)) notFound();

  return {
    messages: (
        await (locale === 'ru'
            ? // When using Turbopack, this will enable HMR for `ru`
            import('../messages/ru.json')
            : import(`../messages/${locale}.json`))
    ).default
  };
});
