import { readFileSync, existsSync } from 'node:fs'
import { dirname, resolve, relative } from 'node:path'
import type { Plugin } from 'vite'

// Inventory modules that Rollup actually retained, including transitive runtime code.
// Workbox's separately generated service worker is inventoried by the packager.
export function sourceInventory(): Plugin {
  return {
    name: 'mediagrap-source-inventory',
    generateBundle(_options, bundle) {
      const packages = new Map<string, { path: string; name: string; version: string }>()
      for (const output of Object.values(bundle)) {
        if (output.type !== 'chunk') continue
        for (const [id, module] of Object.entries(output.modules)) {
          if (module.renderedLength === 0 || !id.includes('/node_modules/')) continue
          let directory = dirname(id.split('?')[0]!)
          while (directory.includes('/node_modules/')) {
            const file = resolve(directory, 'package.json')
            if (existsSync(file)) {
              const pkg = JSON.parse(readFileSync(file, 'utf8'))
              if (pkg.name && pkg.version) {
                const path = relative(process.cwd(), directory)
                packages.set(path, { path, name: pkg.name, version: pkg.version })
                break
              }
            }
            directory = dirname(directory)
          }
        }
      }
      this.emitFile({ type: 'asset', fileName: 'source-inventory.json', source: JSON.stringify([...packages.values()], null, 2) })
    }
  }
}
