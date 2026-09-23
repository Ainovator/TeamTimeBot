"""Regularize the selected Imagegen TT design after user-approved processing."""
from pathlib import Path
import json
import math
import numpy as np
from PIL import Image, ImageDraw

OUT = Path(__file__).resolve().parent
SIZE = 1024
BLUE = (50, 103, 207)
SUPERSAMPLE = 6
STROKE = 120.0
BAR_WIDTH = 360.0
LETTER_HEIGHT = 675.0
OFFSET = (402.0, 90.0)
LEAN = math.tan(math.radians(6))
CORNER = 16.0

def rounded_polygon(vertices, radius=CORNER):
    points = []
    for i, vertex in enumerate(vertices):
        previous = vertices[i - 1]
        following = vertices[(i + 1) % len(vertices)]
        incoming = math.dist(previous, vertex)
        outgoing = math.dist(following, vertex)
        distance = min(radius, incoming / 2, outgoing / 2)
        entry = tuple(vertex[k] + (previous[k] - vertex[k]) * distance / incoming for k in (0, 1))
        leave = tuple(vertex[k] + (following[k] - vertex[k]) * distance / outgoing for k in (0, 1))
        for step in range(17):
            t = step / 16
            points.append(tuple((1-t)**2 * entry[k] + 2*(1-t)*t*vertex[k] + t*t*leave[k] for k in (0, 1)))
    return points

def letter(dx=0.0, dy=0.0):
    vertices = [(0, 0), (BAR_WIDTH, 0), (BAR_WIDTH, STROKE),
                (240, STROKE), (240, LETTER_HEIGHT), (120, LETTER_HEIGHT),
                (120, STROKE), (0, STROKE)]
    return [(x+dx, y+dy) for x, y in rounded_polygon(vertices)]

def capsule(start, end, radius):
    angle = math.atan2(end[1] - start[1], end[0] - start[0])
    points = []
    for center, initial in [(end, angle-math.pi/2), (start, angle+math.pi/2)]:
        for step in range(65):
            a = initial + math.pi * step / 64
            points.append((center[0]+radius*math.cos(a), center[1]+radius*math.sin(a)))
    return points

# Outline of two identical Ts joined at their crossbars. The diagonal bridge
# has perpendicular thickness 168/sqrt(2), matching the 120-unit main strokes.
outline = [(0, 0), (360, 0), (450, 90), (762, 90), (762, 210),
           (642, 210), (642, 765), (522, 765), (522, 210), (402, 210),
           (312, 120), (240, 120), (240, 675), (120, 675), (120, 120), (0, 120)]
shapes = [rounded_polygon(outline)]
shapes = [[(x + LEAN * (LETTER_HEIGHT + OFFSET[1] - y), y) for x, y in shape] for shape in shapes]
all_points = [point for shape in shapes for point in shape]
x0, y0 = map(min, zip(*all_points))
x1, y1 = map(max, zip(*all_points))
scale = SIZE * 0.76 / max(x1-x0, y1-y0)
tx = (SIZE-(x1-x0)*scale)/2 - x0*scale
ty = (SIZE-(y1-y0)*scale)/2 - y0*scale

mask = Image.new('L', (SIZE*SUPERSAMPLE, SIZE*SUPERSAMPLE), 0)
draw = ImageDraw.Draw(mask)
for shape in shapes:
    draw.polygon([((x*scale+tx)*SUPERSAMPLE, (y*scale+ty)*SUPERSAMPLE) for x, y in shape], fill=255)
mask = mask.resize((SIZE, SIZE), Image.Resampling.LANCZOS)
# Remove subvisible resampling fringes without changing the visible contour.
alpha = np.asarray(mask).copy()
alpha[alpha < 3] = 0
alpha[alpha > 252] = 255
mask = Image.fromarray(alpha)
logo = Image.new('RGBA', (SIZE, SIZE), (*BLUE, 0))
logo.putalpha(mask)
rgba = np.asarray(logo).copy()
rgba[alpha == 0, :3] = 0
logo = Image.fromarray(rgba)
destination = OUT / 'teamtime-tt-final-1024.png'
if destination.exists():
    raise FileExistsError(f'Refusing to overwrite {destination}')
logo.save(destination, optimize=True)

pixels = np.asarray(Image.open(destination))
visible = pixels[:, :, 3] > 0
ys, xs = np.where(visible)
colors = np.unique(pixels[visible, :3], axis=0).tolist()
verification = {
    'file': str(destination), 'size': list(logo.size), 'mode': logo.mode,
    'visible_rgb_colors': colors,
    'transparent_pixels': int((pixels[:, :, 3] == 0).sum()),
    'opaque_pixels': int((pixels[:, :, 3] == 255).sum()),
    'visible_bounds_inclusive': [int(xs.min()), int(ys.min()), int(xs.max()), int(ys.max())],
    'margins_px': [int(xs.min()), int(ys.min()), SIZE-1-int(xs.max()), SIZE-1-int(ys.max())],
    'geometry': {'identical_letters': True, 'stroke': STROKE, 'bar_width': BAR_WIDTH,
                 'letter_height': LETTER_HEIGHT, 'right_letter_offset': list(OFFSET),
                 'rightward_lean_degrees': 6, 'corner_radius': CORNER},
    'method': 'Built-in image_gen concept, followed by user-authorized geometric regularization and PNG export.',
}
assert logo.size == (1024, 1024) and colors == [list(BLUE)]
assert verification['transparent_pixels'] > 0 and verification['opaque_pixels'] > 0
assert np.all(pixels[0, :, 3] == 0) and np.all(pixels[-1, :, 3] == 0)
assert np.all(pixels[:, 0, 3] == 0) and np.all(pixels[:, -1, 3] == 0)
assert max(verification['margins_px']) - min(verification['margins_px']) <= 3
(OUT / 'teamtime-tt-verification.json').write_text(json.dumps(verification, indent=2), encoding='utf-8')

# Private visual QA: actual 24 px icons and enlarged copies on light/dark backgrounds.
qa = Image.new('RGB', (960, 620), 'white')
qa_draw = ImageDraw.Draw(qa)
for i, background in enumerate(['#FFFFFF', '#171D2A', '#EDF2F9']):
    left = 320*i
    qa_draw.rectangle((left, 0, left+319, 619), fill=background)
    preview = logo.resize((300, 300), Image.Resampling.LANCZOS)
    qa.paste(preview, (left+10, 12), preview)
    tiny = logo.resize((24, 24), Image.Resampling.LANCZOS)
    qa.paste(tiny, (left+148, 345), tiny)
    enlarged = tiny.resize((192, 192), Image.Resampling.NEAREST)
    qa.paste(enlarged, (left+64, 405), enlarged)
qa.save(OUT / 'teamtime-tt-qa.png')
print(json.dumps(verification, indent=2))
