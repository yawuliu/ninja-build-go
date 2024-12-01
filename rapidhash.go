package main

const RAPID_SEED = (0xbdd89aa982704029)

var rapid_secret = []uint64{0x2d358dccaa6c78a5, 0x8bb84b93962eacc9, 0x4b33a62ed433d4a3}


func rapidhash_internal(key string,  seed uint64, secret []uint64) uint64 {
	len := len(key)
  p := key
  seed^=rapid_mix(seed^secret[0],secret[1])^len
   a := uint64(0)
   b := uint64(0)
  if(_likely_(len<=16)) {
	  if (_likely_(len >= 4)) {
		  const uint8_t *plast = p + len - 4;
		  a = (rapid_read32(p) << 32) | rapid_read32(plast);
		  const uint64_t delta = ((len & 24) >> (len >> 3));
		  b = ((rapid_read32(p+delta) << 32) | rapid_read32(plast-delta));
	  }
     } else if(_likely_(len>0)){
		a=rapid_readSmall(p,len);
		b=0;
	  }else {
		  a = b=0
	  }
  }  else{
    i :=len
    if(_unlikely_(i>48)){
      see1 :=seed
	  see2 :=seed
      while(_likely_(i>=96)){
        seed=rapid_mix(rapid_read64(p)^secret[0],rapid_read64(p+8)^seed);
        see1=rapid_mix(rapid_read64(p+16)^secret[1],rapid_read64(p+24)^see1);
        see2=rapid_mix(rapid_read64(p+32)^secret[2],rapid_read64(p+40)^see2);
        seed=rapid_mix(rapid_read64(p+48)^secret[0],rapid_read64(p+56)^seed);
        see1=rapid_mix(rapid_read64(p+64)^secret[1],rapid_read64(p+72)^see1);
        see2=rapid_mix(rapid_read64(p+80)^secret[2],rapid_read64(p+88)^see2);
        p+=96; i-=96;
      }
      if(_unlikely_(i>=48)){
        seed=rapid_mix(rapid_read64(p)^secret[0],rapid_read64(p+8)^seed);
        see1=rapid_mix(rapid_read64(p+16)^secret[1],rapid_read64(p+24)^see1);
        see2=rapid_mix(rapid_read64(p+32)^secret[2],rapid_read64(p+40)^see2);
        p+=48; i-=48;
      }

      seed^=see1^see2;
    }
    if(i>16){
      seed=rapid_mix(rapid_read64(p)^secret[2],rapid_read64(p+8)^seed^secret[1]);
      if(i>32)
        seed=rapid_mix(rapid_read64(p+16)^secret[2],rapid_read64(p+24)^seed);
    }
    a=rapid_read64(p+i-16);  b=rapid_read64(p+i-8);
  }
  a^=secret[1]
  b^=seed
  rapid_mum(&a,&b)
  return  rapid_mix(a^secret[0]^len,b^secret[1]);
}

func rapidhash_withSeed(key string, seed uint64) uint64 {
	return rapidhash_internal(key,  seed, rapid_secret)
}

func rapidhash(key string) uint64 {
	return rapidhash_withSeed(key, RAPID_SEED)
}
