import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Cuttlegate',
  tagline: 'Feature flags for teams that ship',
  favicon: 'img/favicon.ico',

  future: {
    v4: true,
  },

  url: 'https://the127.github.io',
  baseUrl: '/cuttlegate/',

  organizationName: 'The127',
  projectName: 'cuttlegate',
  trailingSlash: false,

  onBrokenLinks: 'throw',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    colorMode: {
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'Cuttlegate',
      logo: {
        alt: 'Cuttlegate Logo',
        src: 'img/logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'sdkSidebar',
          position: 'left',
          label: 'SDKs',
        },
        {
          href: 'https://github.com/The127/cuttlegate',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'SDKs',
          items: [
            {label: 'Go SDK', to: '/docs/go'},
            {label: 'JavaScript/TypeScript SDK', to: '/docs/js'},
            {label: 'Python SDK', to: '/docs/python'},
          ],
        },
        {
          title: 'More',
          items: [
            {
              label: 'GitHub',
              href: 'https://github.com/The127/cuttlegate',
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Cuttlegate. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['go', 'python', 'bash'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
