# render

Writes the SVGs of this profile and of the HYZIS organization profile
(`hyz-is/.github`), in the HYZIS design system: #000, #FFF and #222, Geist,
pills, 24px cards and 10% hairlines. Colour appears only inside the product
screens. Every image has a light and a dark file, swapped by `<picture>`.

```sh
GOWORK=off go run . -profile paulo -out ../assets
GOWORK=off go run . -profile hyzis -out ../../hyz-is/profile/assets
```

- `screens/`: 960px JPEG captures of the product home pages, embedded in the hero.
- `fonts/`: Geist and Geist Mono (SIL OFL 1.1). The woff2 files are embedded; the TTFs only measure text.
- `icons/`: monochrome marks from Simple Icons 16.35.0 (CC0).
- `brand/`: the Arandu and HYZIS marks.

The hero holds the hyz.is product stage (windows that cycle every 4.8s) and the
Tayi composer typing its prompts. Both stop under `prefers-reduced-motion`.
