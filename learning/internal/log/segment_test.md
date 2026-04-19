# `segment_test.go`

## `TestSegment`

1. Create a temp dir.
2. Configure the segment to roll after **3 entries** (`MaxIndexBytes = entWidth * 3`).
3. Build a segment at `baseOffset = 16`. Verify `nextOffset == 16` and `!IsMaxed()`.
4. Append the same record 3 times; each returned offset should be 16, 17, 18.
5. After the 3rd append, `IsMaxed()` should be true because the index hit its cap.

(The full test also covers reading back, reopening the segment from disk, and `Remove`.)

## Why interesting

This test is the clearest demonstration of segments being the unit of rotation: you can set a tiny `MaxIndexBytes` and watch `IsMaxed` flip exactly when you expect it to.
