"""
Easing functions.

t    current time
b    start value
c    change in value
d    duration
"""

import math


def linear_tween(t, b, c, d):
    """
    No easing, no acceleration.
    """
    return c * t / d + b


def quadratic_ease_in(t, b, c, d):
    """
    Accelerating from zero velocity.
    """
    t /= d
    return c * t * t + b


def quadratic_ease_out(t, b, c, d):
    """
    Decelerating to zero velocity.
    """
    t /= d
    return -c * t * (t - 2) + b


def quadratic_ease_in_ease_out(t, b, c, d):
    """
    Accelerate until halfway, then decelerate.
    """
    t /= d/2
    if t < 1:
        return c / 2 * t * t + b
    t -= 1
    return -c / 2 * (t * (t - 2) - 1) + b


def cubic_ease_in(t, b, c, d):
    """
    Accelerating from zero velocity.
    """
    t /= d
    return c * t * t * t + b


def cubic_ease_out(t, b, c, d):
    """
    Decelerating to zero velocity.
    """
    t /= d
    t -= 1
    return c * (t * t * t + 1) + b


def cubic_ease_in_ease_out(t, b, c, d):
    """
    Accelerate until halfway, then decelerate.
    """
    t /= d/2
    if t < 1:
        return c / 2 * t * t * t + b
    t -= 2
    return c/2 * (t * t * t + 2) + b


def quartic_ease_in(t, b, c, d):
    t /= d
    return c * t * t * t * t + b


def quartic_ease_out(t, b, c, d):
    t /= d
    t -= 1
    return -c * (t * t * t * t - 1) + b


def quartic_ease_in_ease_out(t, b, c, d):
    t /= d/2
    if t < 1:
        return c / 2 * t * t * t * t + b
    t -= 2
    return -c / 2 * (t * t * t * t - 2) + b


if __name__ == '__main__':
    import matplotlib.pyplot as plt

    b = 0
    c = 1
    d = 20

    funs = [
        linear_tween,
        quadratic_ease_in,
        quadratic_ease_out,
        quadratic_ease_in_ease_out,
        cubic_ease_in,
        cubic_ease_out,
        cubic_ease_in_ease_out,
        quartic_ease_in,
        quartic_ease_out,
        quartic_ease_in_ease_out,
    ]

    # plt.figsize = (12, 10)
    fig, axes = plt.subplots(ncols=2, nrows=5)
    for ax, f in zip(axes.flat, funs):
        ts = [f(t, b, c, d) for t in range(d)]
        ax.plot(range(d), ts)
        ax.axis("off")
        # ax.set_title(f.__name__)

    plt.axis("off")
    plt.savefig("easing.png")
    # plt.subplots_adjust(wspace=0.4, hspace=2.0)
    # plt.tight_layout()



