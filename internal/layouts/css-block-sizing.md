# CSS Block Element Sizing in Normal Flow

Based on CSS2 and CSS Sizing Level 3.

---

## Width Algorithm

### Constraint Equation

The following must always hold:

```
margin-left + border-left + padding-left + width + padding-right + border-right + margin-right
= inline-size of containing block
```

### Step 1 — Establish the Containing Block

The containing block's inline size is the **content area** of the nearest block ancestor (excluding its own padding). For the initial containing block it is the viewport width.

### Step 2 — Resolve Percentages

`padding-*` and `margin-*` percentages resolve against the **containing block's inline size** (even vertical ones in horizontal writing modes). `width` percentages resolve against the same value.

### Step 3 — Resolve `auto` Values

`width`, `margin-left`, and `margin-right` can each be `auto` or a definite value:

| `width`    | `margin-left` | `margin-right` | Resolution                                                                                  |
|------------|---------------|----------------|---------------------------------------------------------------------------------------------|
| definite   | definite      | definite       | **Over-constrained**: ignore `margin-right` (or `margin-left` in RTL), solve from equation  |
| `auto`     | any           | any            | Set any `auto` margins to `0`, solve `width` = remaining space (stretch/fill-available)     |
| definite   | `auto`        | definite       | Solve for `margin-left` = remaining space                                                   |
| definite   | definite      | `auto`         | Solve for `margin-right` = remaining space                                                  |
| definite   | `auto`        | `auto`         | **Center**: split remaining space equally between both margins                              |

### Step 4 — Apply `min-width` / `max-width` Clamping

After computing the tentative used width:

1. If `tentative-width > max-width` → use `max-width` and re-run Step 3.
2. If `tentative-width < min-width` → use `min-width` and re-run Step 3.

Initial values: `min-width: 0`, `max-width: none`.

### Step 5 — Intrinsic Size Keywords (CSS Sizing Level 3)

These keywords resolve to a definite value before Step 3:

- **`max-content`** — smallest width where content doesn't overflow given infinite space (all content on one line).
- **`min-content`** — smallest width without overflow, respecting forced breaks (longest unbreakable word).
- **`fit-content`** — `min(max-content, max(min-content, stretch))`.
- **`stretch`** — explicitly requests stretch behavior (equivalent to `width: auto` on a block).

### Step 6 — Aspect Ratio Interaction

If `aspect-ratio` is set and one dimension is `auto`, the resolved width can be derived from the computed height (or vice versa) after the above steps.

### Summary Flowchart

```
Containing block inline-size (CB)
        │
        ▼
Resolve % margins & paddings (vs. CB)
        │
        ▼
width == auto?
  ├── YES → margins auto→0, width = CB − borders − paddings − margins
  └── NO  → margins auto → center or fill remaining = CB − width − borders − paddings
        │
        ▼
Clamp: max-width → re-solve if width > max-width
       min-width → re-solve if width < min-width
        │
        ▼
Used width (final)
```

---

## Height Algorithm

Height works fundamentally differently — the block axis is **not constrained**. The block does not need to fill its containing block's height.

- **Width `auto`** → stretch to fill containing block
- **Height `auto`** → shrink to wrap content

### Step 1 — `height: auto` (the default)

Used height = distance from the **top content edge** to the **bottom edge of the last in-flow child's bottom margin** (after margin collapsing).

Exceptions:
- **Floats are excluded** unless the element establishes a Block Formatting Context (BFC) — e.g. `overflow` is not `visible`, or `display: flow-root`.
- **Absolutely positioned children** are excluded entirely.

### Step 2 — Percentage Heights

A `height` percentage resolves against the **containing block's height**. If the containing block's height is `auto`, the percentage resolves to **`auto`** (it has no effect).

This is why `height: 100%` on a child does nothing unless every ancestor up to a fixed-height root has an explicit height:

```css
html, body { height: 100%; }  /* anchors the chain */
.child     { height: 50%; }   /* now resolves correctly */
```

### Step 3 — Margin Collapsing (vertical axis only)

Adjacent vertical margins **collapse** — they merge into a single margin equal to the largest of the two, not their sum.

**Three collapsing scenarios:**

| Scenario                                                                      | Result                                        |
|-------------------------------------------------------------------------------|-----------------------------------------------|
| Adjacent siblings                                                             | `max(margin-bottom-A, margin-top-B)`          |
| Parent + first child (no border/padding/BFC between them)                    | Parent top margin collapses with child top    |
| Parent + last child (same condition)                                          | Parent bottom margin collapses with child bottom |
| Empty block                                                                   | Its own top and bottom margins collapse       |

**With negative margins:** `max(positives) + min(negatives)` — the most-positive and most-negative values are summed.

**Collapsing is blocked by:** a border, a padding, a BFC boundary, a clearance value, or `overflow` other than `visible`.

### Step 4 — `min-height` / `max-height` Clamping

Same pattern as width:

1. If `tentative-height > max-height` → use `max-height`, re-evaluate.
2. If `tentative-height < min-height` → use `min-height`, re-evaluate.

Initial values: `min-height: 0`, `max-height: none`.

### Step 5 — Aspect Ratio Interaction

If `aspect-ratio` is set and `height` is `auto` but `width` is definite:

```
used-height = used-width / aspect-ratio
```

This is computed **before** min/max clamping.

---

## Width vs Height Comparison

|                                    | `width`                              | `height`                                              |
|------------------------------------|--------------------------------------|-------------------------------------------------------|
| Default `auto` behavior            | Stretch to fill containing block     | Shrink to wrap content                                |
| `%` resolves against               | CB **width** (always)                | CB **height** (only if CB height is definite)         |
| Margin collapsing                  | No                                   | Yes — adjacent vertical margins collapse              |
| Must satisfy constraint equation   | Yes                                  | No                                                    |
| Floats included in `auto`          | N/A                                  | Only if element establishes a BFC                     |

---

## References

- [CSS2 §10.3.3 — Block-level, non-replaced elements in normal flow](https://www.w3.org/TR/CSS2/visudet.html#blockwidth)
- [CSS2 §10.6.3 — Block-level non-replaced elements in normal flow](https://www.w3.org/TR/CSS2/visudet.html#normal-block)
- [CSS Sizing Level 3](https://www.w3.org/TR/css-sizing-3/)
