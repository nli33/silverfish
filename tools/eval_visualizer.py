import chess
import numpy as np
from pathlib import Path
import struct
import tkinter as tk
from tkinter import ttk, filedialog, messagebox

GRID_SIZE = 8
CELL_SIZE = 64
BOARD_SIZE = GRID_SIZE * CELL_SIZE
MARGIN = 30
CANVAS_SIZE = BOARD_SIZE + 2 * MARGIN
FONT = ("Consolas", 12)

PIECE_LETTERS = ["K", "Q", "R", "B", "N", "P"]

FLIP = 0b111000
def feature(perspective, piece_color, piece_type, sq: int) -> int:
    friendly = (piece_color == perspective)
    piece_idx = piece_type + (0 if friendly else 6)  # 0..11
    if perspective == chess.BLACK:
        sq ^= FLIP
    return 64 * piece_idx + sq  # 0..767

class NNUEModel:
    def __init__(self):
        self.loaded = False
        self.num_inputs = None
        self.l1 = None
        self.W_in = None   # shape (L1, NUM_INPUTS)
        self.b_in = None   # shape (L1,)
        self.W_out = None  # shape (1, 2*L1)
        self.b_out = None  # scalar

    def load_from_file(self, path: Path):
        with open(path, "rb") as f:
            magic = f.read(4)
            if magic != b"NNUE":
                raise ValueError("not an NNUE file")
            version_bytes = f.read(4)
            version = struct.unpack("<I", version_bytes)[0]
            assert version == 1
            num_inputs = struct.unpack("<I", f.read(4))[0]
            l1 = struct.unpack("<I", f.read(4))[0]

            rest = f.read()
            floats = np.frombuffer(rest, dtype="<f4")  # little-endian float32

            expected = l1 * num_inputs + l1 + (2 * l1) + 1
            if floats.size != expected:
                raise ValueError(f"unexpected float count in NNUE file. Expected {expected}, got {floats.size}")

            idx = 0
            W_in = floats[idx: idx + (l1 * num_inputs)].reshape((l1, num_inputs))
            idx += l1 * num_inputs
            b_in = floats[idx: idx + l1]
            idx += l1
            W_out = floats[idx: idx + (2 * l1)].reshape((1, 2 * l1))
            idx += 2 * l1
            b_out = floats[idx]
            # done

            self.num_inputs = num_inputs
            self.l1 = l1
            self.W_in = W_in
            self.b_in = b_in
            self.W_out = W_out
            self.b_out = b_out
            self.loaded = True

    def evaluate(self, board: chess.Board) -> tuple[float, float]:
        if not self.loaded:
            raise RuntimeError("model not loaded")

        num_inputs = self.num_inputs
        feats_side = np.zeros(num_inputs, dtype=np.float32)
        feats_opp = np.zeros(num_inputs, dtype=np.float32)

        piece_type_to_idx = {chess.PAWN:0, chess.KNIGHT:1, chess.BISHOP:2,
                             chess.ROOK:3, chess.QUEEN:4, chess.KING:5}

        stm = board.turn  # true=white, false=black

        for sq in chess.SQUARES:
            piece = board.piece_at(sq)
            if piece is None:
                continue
            pt_idx = piece_type_to_idx[piece.piece_type]
            feature_index = feature(stm, piece.color, pt_idx, sq)
            opp_feature_index = feature(not stm, piece.color, pt_idx, sq)
            
            feats_side[feature_index] = 1.0
            feats_opp[opp_feature_index] = 1.0
        
        # np.set_printoptions(threshold=np.inf)
        # print(self.W_in.transpose())

        # W_in shape: (L1, NUM_INPUTS)
        a_side = np.dot(self.W_in, feats_side) + self.b_in
        a_opp  = np.dot(self.W_in, feats_opp)  + self.b_in
        # print(a_side)
        # print("\n\n\n\n\n\n")
        # print(a_opp)
        h_side = np.maximum(0.0, a_side)
        h_opp  = np.maximum(0.0, a_opp)
        concat = np.concatenate([h_side, h_opp])  # shape (2*L1,)
        raw = float(np.dot(self.W_out, concat) + self.b_out)  # in pawns (assumption)
        centipawns = raw * 1000.0
        return raw, centipawns


class NNUEVisualizer(tk.Tk):
    def __init__(self):
        super().__init__()
        self.title("NNUE Evaluation Visualizer")
        self.geometry(f"{CANVAS_SIZE+500}x{CANVAS_SIZE+260}")
        self.resizable(False, False)

        self.board = chess.Board.empty()  # start with empty board
        self.model = NNUEModel()

        # drawing canvas
        self.canvas = tk.Canvas(self, width=CANVAS_SIZE, height=CANVAS_SIZE, bg="#aaa")
        self.canvas.grid(row=0, column=0, columnspan=6, padx=5, pady=5)
        self.canvas.bind("<Button-1>", self.on_click_left)
        self.canvas.bind("<Button-3>", self.on_click_right)

        # FEN entry
        ttk.Label(self, text="FEN:", font=FONT).grid(row=1, column=0, sticky="w", padx=5)
        self.var_fen = tk.StringVar()
        self.entry_fen = ttk.Entry(self, textvariable=self.var_fen, font=FONT, width=80)
        self.entry_fen.grid(row=1, column=1, columnspan=4, sticky="w", padx=5)
        self.entry_fen.bind("<Return>", self.on_fen_enter)
        ttk.Button(self, text="Apply FEN", command=self.on_fen_apply).grid(row=1, column=5, padx=5)

        # NNUE path entry + load
        ttk.Label(self, text="NNUE file:", font=FONT).grid(row=2, column=0, sticky="w", padx=5)
        self.var_nnue_path = tk.StringVar()
        self.entry_nnue = ttk.Entry(self, textvariable=self.var_nnue_path, font=FONT, width=60)
        self.entry_nnue.grid(row=2, column=1, columnspan=3, sticky="w", padx=5)
        ttk.Button(self, text="Browse...", command=self.browse_nnue).grid(row=2, column=4, padx=2)
        ttk.Button(self, text="Load NNUE", command=self.load_nnue).grid(row=2, column=5, padx=2)

        # Piece palette
        self.selected_piece = tk.StringVar(value="P")
        self.selected_color = tk.StringVar(value="white")
        palette_frame = ttk.Frame(self)
        palette_frame.grid(row=3, column=0, columnspan=6, pady=8)
        ttk.Label(palette_frame, text="Piece:", font=FONT).grid(row=0, column=0, padx=4)
        for i, p in enumerate(PIECE_LETTERS):
            b = ttk.Radiobutton(palette_frame, text=p, value=p, variable=self.selected_piece)
            b.grid(row=0, column=1 + i)
        ttk.Label(palette_frame, text="Color:", font=FONT).grid(row=1, column=0, padx=4)
        ttk.Radiobutton(palette_frame, text="White", value="white", variable=self.selected_color).grid(row=1, column=1)
        ttk.Radiobutton(palette_frame, text="Black", value="black", variable=self.selected_color).grid(row=1, column=2)
        ttk.Button(palette_frame, text="Delete Mode (Right-click)", command=lambda: self.selected_piece.set("")).grid(row=1, column=3, padx=8)
        ttk.Button(palette_frame, text="Clear Board", command=self.clear_board).grid(row=1, column=4, padx=8)
        ttk.Button(palette_frame, text="Starting Position", command=self.starting_position).grid(row=1, column=5, padx=8)

        # evaluation display
        eval_frame = ttk.Frame(self)
        eval_frame.grid(row=4, column=0, columnspan=6, pady=6)
        ttk.Label(eval_frame, text="Evaluation:", font=("Consolas", 14, "bold")).grid(row=0, column=0, padx=5)
        self.var_eval = tk.StringVar(value="(no model loaded)")
        ttk.Label(eval_frame, textvariable=self.var_eval, font=("Consolas", 14)).grid(row=0, column=1, padx=5)
        self.var_raw = tk.StringVar(value="")
        ttk.Label(eval_frame, textvariable=self.var_raw, font=("Consolas", 10)).grid(row=1, column=0, columnspan=2)

        self.draw_board()
        self.update_fen()

    def browse_nnue(self):
        path = filedialog.askopenfilename(filetypes=[("NNUE files", "*.nnue"), ("All files","*.*")])
        if path:
            self.var_nnue_path.set(path)

    def load_nnue(self):
        path = self.var_nnue_path.get().strip()
        if not path:
            messagebox.showwarning("No file", "Please select a .nnue file path first.")
            return
        try:
            self.model.load_from_file(Path(path))
            messagebox.showinfo("Loaded", f"Loaded NNUE (NUM_INPUTS={self.model.num_inputs}, L1={self.model.l1})")
            self.update_evaluation()
        except Exception as e:
            messagebox.showerror("Load error", f"Failed to load NNUE: {e}")

    def clear_board(self):
        self.board = chess.Board.empty()
        self.draw_board()
        self.update_fen()
        self.update_evaluation()

    def starting_position(self):
        self.board = chess.Board()  # standard starting pos
        self.draw_board()
        self.update_fen()
        self.update_evaluation()

    def on_fen_enter(self, event):
        self.on_fen_apply()

    def on_fen_apply(self):
        fen_text = self.var_fen.get().strip()
        if not fen_text:
            return
        try:
            # Allow user to input only piece placement or full FEN.
            # chess.Board accepts full FEN. If just piece placement, append default fields.
            if " " not in fen_text:
                fen_text_full = fen_text + " w - - 0 1"
            else:
                fen_text_full = fen_text
            b = chess.Board(fen=fen_text_full)
            # Keep exactly the placement, turn, etc. but our canvas only displays piece placement
            # we set board to b
            self.board = b
            self.draw_board()
            self.update_fen()
            self.update_evaluation()
        except Exception as e:
            messagebox.showerror("FEN error", f"Could not parse FEN: {e}")

    def update_fen(self):
        # produce full FEN string from self.board
        try:
            fen = self.board.fen()
            self.var_fen.set(fen)
        except Exception:
            self.var_fen.set("")

    def square_from_coords(self, x, y):
        clicked_x = x - MARGIN
        clicked_y = y - MARGIN
        c = int(clicked_x // CELL_SIZE)
        r = int(clicked_y // CELL_SIZE)
        if 0 <= c < GRID_SIZE and 0 <= r < GRID_SIZE:
            file = c
            rank = 7 - r
            sq = chess.square(file, rank)
            return sq
        return None

    def on_click_left(self, event):
        sq = self.square_from_coords(event.x, event.y)
        if sq is None:
            return
        sel = self.selected_piece.get()
        if sel:
            color = chess.WHITE if self.selected_color.get() == "white" else chess.BLACK
            # map letter to piece type
            letter = sel.upper()
            piece_map = {"P": chess.PAWN, "N": chess.KNIGHT, "B": chess.BISHOP,
                         "R": chess.ROOK, "Q": chess.QUEEN, "K": chess.KING}
            piece = chess.Piece(piece_map[letter], color)
            # place piece (replace whatever is there)
            self.board.set_piece_at(sq, piece)
        else:
            # if no piece selected, toggle through removing/placing? We'll toggle empty -> nothing
            self.board.remove_piece_at(sq)
        self.draw_board()
        self.update_fen()
        self.update_evaluation()

    def on_click_right(self, event):
        # delete piece on right-click
        sq = self.square_from_coords(event.x, event.y)
        if sq is None:
            return
        self.board.remove_piece_at(sq)
        self.draw_board()
        self.update_fen()
        self.update_evaluation()

    def draw_board(self):
        self.canvas.delete("all")
        # squares
        for r in range(GRID_SIZE):
            for c in range(GRID_SIZE):
                x0, y0 = c*CELL_SIZE + MARGIN, r*CELL_SIZE + MARGIN
                x1, y1 = x0+CELL_SIZE, y0+CELL_SIZE
                color = "#eee" if (r+c)%2==0 else "#555"
                self.canvas.create_rectangle(x0, y0, x1, y1, fill=color, outline="")
                # piece?
                file = c
                rank = 7 - r
                sq = chess.square(file, rank)
                piece = self.board.piece_at(sq)
                if piece:
                    # draw letter
                    letter = piece.symbol()
                    # letter returns lowercase for black, uppercase for white
                    # we draw uppercase on canvas and prefix color via fill
                    display = letter.upper()
                    fill = "white" if piece.color == chess.WHITE else "black"
                    self.canvas.create_text((x0+x1)//2, (y0+y1)//2, text=display, font=("Consolas", 28, "bold"), fill=fill)

        # column labels
        for c in range(GRID_SIZE):
            x = c*CELL_SIZE + CELL_SIZE//2 + MARGIN
            y = MARGIN//2
            self.canvas.create_text(x, y, text=chr(ord('A') + c), font=FONT, fill="black")
        # row labels
        for r in range(GRID_SIZE):
            x = MARGIN // 2
            y = r*CELL_SIZE + CELL_SIZE//2 + MARGIN
            self.canvas.create_text(x, y, text=str(8 - r), font=FONT, fill="black")

    def update_evaluation(self):
        if not self.model.loaded:
            self.var_eval.set("(no model loaded)")
            self.var_raw.set("")
            return
        try:
            raw_pawns, cp = self.model.evaluate(self.board)
            # show centipawns rounded
            if abs(cp) > 100000:
                eval_str = f"{cp:.1f} cp"
            else:
                eval_str = f"{int(round(cp))} cp"
            self.var_eval.set(eval_str)
            self.var_raw.set(f"Raw (pawns): {raw_pawns:.6f}   (converted: {cp:.2f} cp)")
        except Exception as e:
            self.var_eval.set("(eval error)")
            self.var_raw.set(str(e))

if __name__ == "__main__":
    app = NNUEVisualizer()
    app.mainloop()
