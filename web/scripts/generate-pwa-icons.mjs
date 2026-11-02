// Regenerates PWA / app icons from public/favicon.svg into public/.
// Run with: pnpm generate-icons
//
// InkDock ships a 1024x1024 vector favicon.svg with a warm tile behind the mark.
// Standard PWA icons composite it onto an opaque dark tile, and the maskable
// variant adds an 80% safe zone. iOS supplies the Apple icon's outer corners.
import { readFile, writeFile } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import sharp from 'sharp'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const publicDir = path.resolve(__dirname, '..', 'public')
const svgPath = path.join(publicDir, 'favicon.svg')

// Keep generated install icons opaque so home-screen tiles never show fringes.
const BACKGROUND = { r: 26, g: 26, b: 26 }

async function renderAppleTouchIcon() {
  // iOS supplies the outer mask, so extend the warm inset tile to the canvas.
  const svg = (await readFile(svgPath, 'utf8'))
    .replaceAll('x="80" y="80" width="864" height="864" rx="188"',
      'x="0" y="0" width="1024" height="1024"')
  await sharp(Buffer.from(svg))
    .resize(180, 180)
    .removeAlpha()
    .png()
    .toFile(path.join(publicDir, 'apple-touch-icon.png'))
  console.log('  ✓ apple-touch-icon.png (180x180, full-bleed)')
}

async function renderIcon(size, file) {
  const layer = await sharp(svgPath).resize(size, size).png().toBuffer()
  await sharp({ create: { width: size, height: size, channels: 4, background: BACKGROUND } })
    .composite([{ input: layer, blend: 'over' }])
    .png()
    .toFile(path.join(publicDir, file))
  console.log(`  ✓ ${file} (${size}x${size})`)
}

async function renderMaskable(size, file) {
  // Maskable icons reserve a center 80% "safe zone"; pad the artwork so
  // platform-chrome cropping never clips the book motif.
  const inner = Math.round(size * 0.8)
  const offset = Math.round((size - inner) / 2)
  const layer = await sharp(svgPath).resize(inner, inner).png().toBuffer()
  await sharp({ create: { width: size, height: size, channels: 4, background: BACKGROUND } })
    .composite([{ input: layer, blend: 'over', top: offset, left: offset }])
    .png()
    .toFile(path.join(publicDir, file))
  console.log(`  ✓ ${file} (${size}x${size}, maskable safe-zone)`)
}

async function renderMaskIcon() {
  const svg = (await readFile(svgPath, 'utf8'))
    .replace(/\s*<rect x="80" y="80" width="864" height="864" rx="188" fill="#[0-9a-fA-F]{6}"\/>/, '')
    .replace(/fill="#[0-9a-fA-F]{6}"/g, 'fill="black"')
  await writeFile(path.join(publicDir, 'mask-icon.svg'), svg, 'utf8')
  console.log('  ✓ mask-icon.svg (monochrome pinned-tab icon)')
}

async function main() {
  try {
    await readFile(svgPath)
  } catch {
    console.error(`favicon.svg not found at ${svgPath}`)
    process.exit(1)
  }
  console.log('Generating InkDock PWA icons from favicon.svg…')
  await renderAppleTouchIcon()
  await renderIcon(192, 'pwa-192.png')
  await renderIcon(512, 'pwa-512.png')
  await renderMaskable(512, 'pwa-maskable-512.png')
  await renderMaskIcon()
  console.log('Done.')
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
