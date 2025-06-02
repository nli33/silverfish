import tkinter as tk
from tkinter import ttk

GRID_SIZE = 8
CELL_SIZE = 64
BOARD_SIZE = GRID_SIZE * CELL_SIZE
FONT = ("Consolas", 12)

class BitboardVisualizer(tk.Tk):
    def __init__(self):
        super().__init__()
        self.title("Chess Bitboard Visualizer")
        self.bitboard = 0

        # Canvas for board visualization
        self.canvas = tk.Canvas(self, width=BOARD_SIZE, height=BOARD_SIZE, bg="#aaa")
        self.canvas.grid(row=0, column=0, columnspan=3)
        self.canvas.bind("<Button-1>", self.on_click)

        # StringVars for entries
        self.var_bin = tk.StringVar()
        self.var_hex = tk.StringVar()
        self.var_dec = tk.StringVar()

        entries = [
            ("Binary", self.var_bin),
            ("Hex", self.var_hex),
            ("Decimal", self.var_dec)
        ]
        for i, (label, var) in enumerate(entries):
            ttk.Label(self, text=label, font=FONT).grid(row=1+i, column=0, sticky="w", padx=5)
            entry = ttk.Entry(self, textvariable=var, font=FONT, width=80)
            entry.grid(row=1+i, column=1, columnspan=2, sticky="ew", padx=5, pady=2)
            entry.bind("<Return>", self.on_enter)

        # Initial draw
        self.draw_board()
        self.update_entries()

    def draw_board(self):
        self.canvas.delete("all")
        for r in range(GRID_SIZE):
            for c in range(GRID_SIZE):
                x0, y0 = c*CELL_SIZE, r*CELL_SIZE
                x1, y1 = x0+CELL_SIZE, y0+CELL_SIZE
                color = "#eee" if (r+c)%2==0 else "#555"
                self.canvas.create_rectangle(x0, y0, x1, y1, fill=color, outline="")
                index = (7-r)*8 + c
                if (self.bitboard >> index) & 1:
                    self.canvas.create_rectangle(x0+4, y0+4, x1-4, y1-4, fill="#3498db", outline="")

    def format_binary(self, value: int) -> str:
        bits = format(value, '064b')
        groups = [bits[i:i+8] for i in range(0, 64, 8)]
        return '0b' + '_'.join(groups)

    def on_click(self, event):
        c = event.x // CELL_SIZE
        r = event.y // CELL_SIZE
        if 0 <= c < GRID_SIZE and 0 <= r < GRID_SIZE:
            index = (7-r)*8 + c
            self.bitboard ^= (1 << index)
            self.draw_board()
            self.update_entries()

    def update_entries(self):
        # Update entries based on current bitboard
        self.var_bin.set(self.format_binary(self.bitboard))
        self.var_hex.set('0x' + format(self.bitboard, '016x'))
        self.var_dec.set(str(self.bitboard))

    def on_enter(self, event):
        # Parse on Return pressed, then reformat all fields
        widget = event.widget
        content = widget.get().strip()
        try:
            if widget is self.nametowidget(str(widget)) and widget.getvar(widget.cget('textvariable')) == self.var_bin.get():
                # Binary field
                text = content
                if text.lower().startswith('0b'):
                    text = text[2:]
                text = text.replace('_', '')
                new = int(text, 2)
            elif content.lower().startswith('0x') or content.isalnum() and all(c in '0123456789abcdefABCDEF' for c in content):
                # Hex field
                text = content
                if text.lower().startswith('0x'):
                    text = text[2:]
                new = int(text, 16)
            else:
                # Decimal field
                new = int(content)
            self.bitboard = new & ((1 << 64) - 1)
        except ValueError:
            pass
        self.draw_board()
        self.update_entries()

if __name__ == '__main__':
    app = BitboardVisualizer()
    app.mainloop()

