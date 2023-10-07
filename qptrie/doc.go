// Package qptrie defines an implementation of a QP-Trie data structure with opinionated
// extensions.
//
// A QP-Trie consists of a number of connected Twigs (nodes and leaves). All branches
// end with a leaf Twig.
//
// Each Twig has two fields:
// ------------------------
//
//   - bitpack - 64-bit packed settings of the twig (the structure depends on a twig type);
//   - pointer - an unsafe.Pointer to a child Twig (node/leaf) or a value (KV/raw).
//
// Bitpack structure variants:
// --------------------------
//
//   - Regular Leaf:
//
//     [ 1:63 ] [ 1:62] [ 3:61-59 ] [           59:58-00            ]
//     <1:leaf> <0:reg> <SSS:shift> ---------------------------------
//
//   - Embedding Leaf:
//
//     [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [    56:55-00     ]
//     <1:leaf> <1:emb> <SSS:shift> <NNN:emb-len> <KKK...KKK:emb-key>
//
//   - Fan-node:
//
//     [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [    6:55-50    ] [  49-..  ] [ 32|16|08|04|02-00 ]
//     <0:node> <0:fan> <SSS:shift> <NNN:nib-len> <PPPPPP:pfx-size> <K...K:pfx> <BBB...BBBB:twig-bitmap>
//
//     pfx-size is [0..47|45|41|33|17] (max depends on nib-len)
//
//     nib-len is [1|2|3|4|5]
//
//     nib len  nib range  pfx range                  bitmap
//     -------  ---------  ---------  ---------------------------------------
//     001 = 1    [0..1]    [0..47]   1+2:                                000
//     010 = 2    [0..3]    [0..45]   1+4:                              00000
//     011 = 3    [0..7]    [0..41]   1+8:                          000000000
//     100 = 4   [0..15]    [0..33]   1+16:                 00000000000000000
//     101 = 5   [0..31]    [0..17]   1+32: 000000000000000000000000000000000
//
//     One extra bit in the bitmap is needed to differentiate an empty nib and genuine zero bits.
//
//   - Regular Cut-Node:
//
//     [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [    56:55-00     ]
//     <0:node> <1:cut> <SSS:shift> <000:not-emb> -------------------
//
//   - Embedding Cut-Node:
//
//     [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [    56:55-00     ]
//     <0:node> <1:cut> <SSS:shift> <NNN:emb-len> [KKK...KKK:emb-key]
//
// Pointer variants:
// ----------------
//
//   - Regular Leaf:        unsafe.Pointer( &KV{Key:"tail", Val:any} )
//   - Embedding Leaf:      unsafe.Pointer( *any )
//   - Fan-Node:            unsafe.Pointer( *[N]Twig )
//   - Regular Cut-Node:    unsafe.Pointer( &KV{Key:"tail", Val:*Twig} )
//   - Embedding Cut-Node:  unsafe.Pointer( *Twig )
//
// Example trie:
// ------------
//
//			              ,-- [leaf:"var/log/syslog"]
//	                  |
//		[fan:pfx="/"] --+-- [cut:"home/"] -- [fan:pfx="user1/"] -- [leaf:"tmp/1.txt"]
//		                |
//		                |                               ,-- [leaf:"bash"]
//			              `-- [cut:"usr/bin/"] -- [fan] --+
//		                                                `-- [leaf:"vim"]
//
// The trie above contains the following keys:
//
//   - "/var/log/syslog"
//   - "/home/user1/tmp/1.txt"
//   - "/usr/bin/bash"
//   - "/usr/bin/vim"
package qptrie
