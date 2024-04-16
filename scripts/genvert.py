import time

KEY_ALPHABET = [
    chr(x) for x in range(ord('a'), ord('z') + 1)] + [chr(x) for x in range(ord('A'), ord('Z') + 1)] + \
    ['%d' % i for i in range(10)]


def encode_num(hex_num):
    ans = []
    while hex_num > 0 and len(ans) < 8:
        p = hex_num % len(KEY_ALPHABET)
        ans.append(KEY_ALPHABET[p])
        hex_num = int(hex_num / len(KEY_ALPHABET))
    return ''.join([str(x) for x in ans])

if __name__ == '__main__':
    print('<doc id="foo">')
    p_idx = 1
    for i in range (1, 21):
        if (i-1) % 10 == 0:
            if p_idx > 1:
                print('</p>')
            print(f'<p id="par{p_idx}">')
            p_idx += 1
        print('{}\t{}\tdata:{}'.format(i, encode_num(i), i))
        if i % 5 == 0:
            print('<nl />')
        if i % 5 == 2:
            print('<m/>')
        time.sleep(0.004)
    print('</doc>')
