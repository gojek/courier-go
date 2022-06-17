// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

const lightCodeTheme = require('prism-react-renderer/themes/github');
const darkCodeTheme = require('prism-react-renderer/themes/dracula');

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'Courier Go',
  tagline: 'Information Superhighway',
  url: 'https://gojek.github.io',
  baseUrl: '/courier-go/',
  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',
  favicon: 'img/favicon.ico',

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: 'gojek', // Usually your GitHub org/user name.
  projectName: 'courier-go', // Usually your repo name.

  // Even if you don't use internalization, you can use this field to set useful
  // metadata like html lang. For example, if your site is Chinese, you may want
  // to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          sidebarPath: require.resolve('./sidebars.js'),
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          editUrl:
            'https://github.com/facebook/docusaurus/tree/main/packages/create-docusaurus/templates/shared/',
        },
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      navbar: {
        title: 'Courier Go',
        logo: {
          alt: 'Courier Logo',
          src: 'img/courier-logo.svg',
        },
        items: [
          {
            type: 'doc',
            docId: 'getting-started',
            position: 'left',
            label: 'Docs',
          },
          {
            type: 'doc',
            docId: 'Tutorials/connect',
            position: 'left',
            label: 'Tutorials',
          },
          {
            href: 'https://pkg.go.dev/github.com/gojek/courier-go',
            position: 'left',
            label: 'API',
          },
          {
            href: 'https://github.com/gojek/courier-go',
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Docs',
            items: [
              {
                label: 'Getting Started',
                to: '/docs/getting-started',
              },
              {
                label: 'Tutorials',
                to: '/docs/Tutorials/connect',
              },
              {
                label: 'API',
                href: 'https://pkg.go.dev/github.com/gojek/courier-go',
              },
            ],
          },
          {
            title: 'Community',
            items: [
              { label: 'Gojek open source', href: 'https://github.com/gojek/', },
              { label: 'Discord', href: 'https://discord.gg/C823qK4AK7', },
              { label: 'Twitter', href: 'https://twitter.com/gojektech', },
            ],
          },
          {
            title: 'More',
            items: [
              { label: 'Courier', href: 'https://gojek.github.io/courier/', },
              { label: 'E2E example', href: 'https://gojek.github.io/courier/docs/Introduction', },
              { label: 'Blogs', href: 'https://gojek.github.io/courier/blog', },
              { label: 'Github', href: 'https://github.com/gojek/courier-go', },
            ],
          },
        ],
        logo: {
          alt: 'Gojek Open Source Logo',
          src: 'img/gojek-logo-white.png',
          width: 250,
          height: 35,
          href: 'https://github.com/gojek/',
        },
        copyright: `Copyright © ${new Date().getFullYear()} Gojek. Built with Docusaurus.`,
      },
      prism: {
        theme: lightCodeTheme,
        darkTheme: darkCodeTheme,
      },
      metadata: [
        {},
      ]
    }),
};

module.exports = config;
