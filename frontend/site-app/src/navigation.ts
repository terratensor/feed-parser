import {createLocalizedPathnamesNavigation} from 'next-intl/navigation';
import {pathnames, locales, localePrefix} from './config';

export const {Link, redirect, usePathname, useRouter} =
    createLocalizedPathnamesNavigation({
      locales,
      pathnames,
      localePrefix
    });
