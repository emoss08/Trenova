# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  The Monta software is licensed under the Business Source License 1.1. You are granted the right -
#  to copy, modify, and redistribute the software, but only for non-production use or with a total -
#  of less than three server instances. Starting from the Change Date (November 16, 2026), the     -
#  software will be made available under version 2 or later of the GNU General Public License.     -
#  If you use the software in violation of this license, your rights under the license will be     -
#  terminated automatically. The software is provided "as is," and the Licensor disclaims all      -
#  warranties and conditions. If you use this license's text or the "Business Source License" name -
#  and trademark, you must comply with the Licensor's covenants, which include specifying the      -
#  Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use     -
#  Grant, and not modifying the license in any other way.                                          -
# --------------------------------------------------------------------------------------------------

import librosa
import pygame
import numpy as np
import math
import random
import noise

file_path = "mr-professor.mp3"


# Particle class for snowflakes
class Snowflake:
    def __init__(self, screen_width, screen_height):
        self.x = random.randint(0, screen_width)
        self.y = random.randint(-50, -10)  # Start above the screen
        self.size = random.randint(2, 5)
        self.base_speed = random.uniform(0.5, 1.5)
        self.speed = self.base_speed
        self.color = (255, 255, 255)  # Snowflakes are white

    def fall(self, bass_intensity):
        # Increase speed based on bass intensity
        self.speed = self.base_speed + bass_intensity * 0.5
        self.y += self.speed
        if self.y > screen_height:
            self.reset_position()

    def reset_position(self):
        # Reset snowflake to the top of the screen
        self.x = random.randint(0, screen_width)
        self.y = random.randint(-50, -10)
        self.size = random.randint(2, 5)
        self.base_speed = random.uniform(0.5, 1.5)
        self.speed = self.base_speed

    def draw(self, screen):
        pygame.draw.circle(screen, self.color, (int(self.x), int(self.y)), self.size)


def draw_reflection(screen, snowflakes, water_level, bass_intensity):
    # Modified reflection to shimmer based on bass intensity
    for snowflake in snowflakes:
        inverted_y = water_level + (water_level - snowflake.y)
        alpha = 80 + random.randint(-20, 20) * bass_intensity  # Shimmer effect
        reflection_color = (
            *snowflake.color,
            max(0, min(alpha, 255)),
        )  # Ensure alpha stays in bounds
        if inverted_y >= water_level:
            reflection_surface = pygame.Surface(
                (snowflake.size * 2, snowflake.size * 2), pygame.SRCALPHA
            )
            pygame.draw.circle(
                reflection_surface,
                reflection_color,
                (snowflake.size, snowflake.size),
                snowflake.size,
            )
            screen.blit(
                reflection_surface,
                (snowflake.x - snowflake.size, inverted_y - snowflake.size),
            )


def draw_gradient_sky(screen, top_color, bottom_color):
    for i in range(screen.get_height()):
        ratio = i / screen.get_height()
        gradient_color = [
            top_color[j] * (1 - ratio) + bottom_color[j] * ratio for j in range(3)
        ]
        pygame.draw.line(screen, gradient_color, (0, i), (screen.get_width(), i))


def draw_mountain_silhouette(screen, mountain_color, mountain_highlight, water_level):
    mountain_points = []  # Hold points for drawing mountain peaks
    peak_height = 80
    noise_offset = (
        pygame.time.get_ticks() / 1000
    )  # Change over time for dynamic texture

    for i in range(0, screen.get_width(), 60):
        # Calculate noise value for peak variation
        noise_value = noise.pnoise1((i + noise_offset) * 0.1, repeat=screen.get_width())
        adjusted_peak_height = (
            peak_height + noise_value * 15
        )  # Add noise to the peak height

        peak = (i + 30, water_level - adjusted_peak_height)
        base_left = (i, water_level)
        base_right = (i + 60, water_level)
        mountain_points += [base_left, peak, base_right]

        # Draw highlight on the left side of the peak
        pygame.draw.polygon(
            screen,
            mountain_highlight,
            [peak, base_left, (i + 15, water_level - adjusted_peak_height / 3)],
        )

        # Add shadow to the right side of the peak for depth
        shadow_color = [int(c * 0.6) for c in mountain_color]  # Darker shade for shadow
        pygame.draw.polygon(
            screen,
            shadow_color,
            [peak, base_right, (i + 45, water_level - adjusted_peak_height / 3)],
        )

    # Draw the mountain silhouette with adjusted peaks
    pygame.draw.polygon(
        screen,
        mountain_color,
        mountain_points + [(screen.get_width(), water_level), (0, water_level)],
    )


def create_water_reflection(screen, water_level, bass_intensity):
    # Simulate flowing water reflection
    water_level = int(water_level)
    water_color = (255, 255, 255, 120)
    for x in range(0, screen.get_width(), 20):
        for y in range(water_level, screen.get_height(), 20):
            amplitude = bass_intensity * 5
            frequency = 0.2
            phase = pygame.time.get_ticks() / 1000
            # Apply a sine wave for the flowing water effect
            offset = int(math.sin(frequency * x + phase) * amplitude)
            pygame.draw.circle(screen, water_color, (x, y + offset), 3)


def draw_starry_sky(screen, bass_intensity):
    for i in range(100):  # Number of stars
        x = random.randrange(0, screen.get_width())
        y = random.randrange(0, screen.get_height() // 2)
        star_intensity = random.randint(50, 255)
        pygame.draw.circle(
            screen, (star_intensity, star_intensity, star_intensity), (x, y), 1
        )


def draw_misty_mountains(screen, mountain_color, water_level):
    # Add a mist effect at the base of the mountains
    mist_height = 20  # The height of the mist effect
    for i in range(0, screen.get_width(), 2):
        mist_intensity = random.randint(50, 150)
        pygame.draw.line(
            screen,
            (mist_intensity, mist_intensity, mist_intensity, 50),
            (i, water_level),
            (i, water_level + mist_height),
        )


def draw_snowflakes(screen, snowflakes, bass_intensity):
    for snowflake in snowflakes:
        snowflake.fall(bass_intensity)
        # Rotate the snowflake as it falls
        angle = pygame.time.get_ticks() / 600
        size = snowflake.size
        # Create a new surface to apply rotation
        snowflake_surface = pygame.Surface((size * 2, size * 2), pygame.SRCALPHA)
        pygame.draw.circle(snowflake_surface, snowflake.color, (size, size), size)
        # Rotate and blit the snowflake
        rotated_snowflake = pygame.transform.rotate(snowflake_surface, angle)
        screen.blit(rotated_snowflake, (snowflake.x - size, snowflake.y - size))


# Initialize Pygame and set up the display
pygame.init()
screen_width, screen_height = 1920, 1080
fullscreen = False  # Set to True for fullscreen mode
if fullscreen:
    screen = pygame.display.set_mode((screen_width, screen_height), pygame.FULLSCREEN)
else:
    screen = pygame.display.set_mode((screen_width, screen_height))
pygame.display.set_caption("Music-Reactive Scenery")

# Load audio data
y, sr = librosa.load(file_path, sr=None)
S = np.abs(librosa.stft(y))
bass_frequencies = S[:10, :]

# Start playing music
pygame.mixer.music.load(file_path)
pygame.mixer.music.play()

# Create snowflakes and set up font for text
num_snowflakes = 200
snowflakes = [Snowflake(screen_width, screen_height) for _ in range(num_snowflakes)]
font = pygame.font.SysFont(None, 48)

# Water level for reflection
water_level = screen_height * 0.75

# Main loop
running = True
while running:
    for event in pygame.event.get():
        if event.type == pygame.QUIT:
            running = False

    # Synchronize with music
    current_time = pygame.mixer.music.get_pos() / 1000
    frame_index = int(current_time * sr // 512)
    frame = bass_frequencies[:, frame_index % bass_frequencies.shape[1]]
    bass_intensity = np.mean(frame)

    # Draw gradient sky
    draw_gradient_sky(screen, (15, 24, 42), (26, 35, 64))  # Dark blue to lighter blue

    draw_starry_sky(screen, bass_intensity)
    # Draw snowflakes with glow
    draw_snowflakes(screen, snowflakes, bass_intensity)

    # Draw mountain silhouette
    mountain_color = (33, 33, 60)
    mountain_highlight = (40, 40, 70)
    draw_mountain_silhouette(screen, mountain_color, mountain_highlight, water_level)
    # draw_misty_mountains(screen, mountain_color, water_level)

    # create_water_reflection(screen, water_level, bass_intensity)

    draw_reflection(screen, snowflakes, water_level, bass_intensity)

    pygame.display.flip()
    pygame.time.wait(16)  # 60 FPS

pygame.quit()
