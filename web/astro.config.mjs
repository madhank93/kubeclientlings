// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import sitemap from '@astrojs/sitemap';

// Served from the custom domain at the root.
// (For GitHub project pages instead, set SITE to the github.io origin and BASE to '/kubeclientlings'.)
const SITE = 'https://kubeclientlings.madhan.app';
const BASE = '/';
const DESCRIPTION =
	'Learn Kubernetes client-go the rustlings way — 52 hands-on exercises you fix one at a time, from clientset setup to controller-runtime operators, verified against a real kind cluster.';

export default defineConfig({
	site: SITE,
	base: BASE,
	integrations: [
		sitemap(),
		starlight({
			title: 'KubeClientlings',
			description: DESCRIPTION,
			customCss: ['./src/styles/hero.css'],
			social: [
				{ icon: 'github', label: 'GitHub', href: 'https://github.com/madhank93/kubeclientlings' },
			],
			// SEO / social-share metadata applied to every page.
			head: [
				{ tag: 'meta', attrs: { property: 'og:type', content: 'website' } },
				{ tag: 'meta', attrs: { property: 'og:site_name', content: 'KubeClientlings' } },
				{ tag: 'meta', attrs: { property: 'og:image', content: new URL(`${BASE}favicon.svg`, SITE).href } },
				{ tag: 'meta', attrs: { name: 'twitter:card', content: 'summary' } },
				{ tag: 'meta', attrs: { name: 'twitter:image', content: new URL(`${BASE}favicon.svg`, SITE).href } },
				{ tag: 'meta', attrs: { name: 'theme-color', content: '#326ce5' } },
				{ tag: 'meta', attrs: { name: 'keywords', content: 'Kubernetes, client-go, controller-runtime, operator, rustlings, exercises, kind, informers, CRD' } },
			],
			sidebar: [
				{ label: 'Getting started', slug: 'getting-started' },
				{ label: 'Catalog', link: '/catalog/' },
			],
		}),
	],
});
