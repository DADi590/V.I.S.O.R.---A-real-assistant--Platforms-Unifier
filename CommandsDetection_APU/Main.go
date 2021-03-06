/*
 * Copyright 2021 DADi590
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package CommandsDetection_APU is the submodule that detects commands in a given string of words
package CommandsDetection_APU

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"Assist_Platforms_Unifier/GlobalUtilsInt_APU"
)

const ERR_CMD_DETECT string = GlobalUtilsInt_APU.MOD_RET_ERR_PREFIX + "CMD_DETECT - "

/*
Main is the function to call to request a detection of commands in a given sentence of words.

-----------------------------------------------------------

> Params:

- sentence_str – a sentence of words, for example coming directly from speech recognition

- allowed_cmds – a string containing a list of the CMD_-started constants of all the commands that are allowed to be
returned if found on the 'sentence', separated by CMDS_SEPARATOR - if all commands are wanted for detection, consider
using the return of GenerateListAllCmds()


> Returns:

- a list of the detected commands in the form
	[CMD1][separator][CMD2][separator][CMD3]
with CMDS_SEPARATOR as the separator; if the function detected no commands, an empty string; if any error occurred, a
string beginning with ERR_CMD_DETECT, followed either by GlobalUtils_APU.APU_ERR_PREFIX and its requirements, or a Go
error
*/
func Main(sentence_str string, allowed_cmds string) string {
	var ret_var string

	GlobalUtilsInt_APU.Tcf{
		Try: func() {
			ret_var = MainInternal(sentence_str, allowed_cmds)
		},
		Catch: func(e GlobalUtilsInt_APU.Exception) {
			ret_var = ERR_CMD_DETECT + fmt.Sprint(e)
		},
	}.Do()

	return ret_var
}

const CMDS_SEPARATOR string = ", "

/*
MainInternal is the actual function that will do what's written on Main() - continue reading there.

There is just one exception, which is this one doesn't return any error code if anything goes wrong - it will panic
instead (no protection here), so always call the other one in production code.

Note: if you find this function exported, know it's just for testing from the main package. Do NOT use it in production.
*/
func MainInternal(sentence_str string, allowed_cmds_str string) string {
	sentence_str = sentenceCorrection(sentence_str)

	var sentence []string = strings.Split(sentence_str, " ")
	var allowed_cmds []int = nil
	for _, cmd_index := range strings.Split(allowed_cmds_str, CMDS_SEPARATOR) {
		number, _ := strconv.Atoi(cmd_index)
		allowed_cmds = append(allowed_cmds, number)
	}

	// Prepare the sentence for the NLP analysis.
	sentence_str = sentenceNLPPreparation(&sentence, true)
	// Analyze the sentence with NLP help and, for example, replace all the "it"s on the sentence with their meaning.
	nlpAnalyzer(&sentence, sentence_str)
	// Prepare the sentence for the NLP analysis.
	sentence_str = sentenceNLPPreparation(&sentence, false)

	log.Println(sentence)

	// Get all the commands present on the sentence (according to the allowed_cmds).
	var sentence_cmds []float32 = sentenceCmdsDetector(sentence, allowed_cmds)

	// Filter the sentence of special commands (like "don't"/"do not") and do the necessary for each special command.
	taskFilter(&sentence_cmds)

	var ret_var string = ""
	for _, command := range sentence_cmds {
		ret_var += fmt.Sprint(command) + CMDS_SEPARATOR
	}

	log.Println("::::::::::::::::::::::::::::::::::")
	log.Println(ret_var)

	if strings.HasSuffix(ret_var, CMDS_SEPARATOR) {
		ret_var = ret_var[:len(ret_var)-len(CMDS_SEPARATOR)]
	}

	// Remove consecutively repeated commands
	//ret_var = removeRepeatedCmds(ret_var) - let's see if the verification function can handle it without this...

	log.Println(ret_var)
	log.Println("::::::::::::::::::::::::::::::::::")

	return ret_var
}

/*
removeRepeatedCmds removes immediately repeated commands from the commands verification ([1, 3, 3, 4, 3, 4] will become
[1, 3, 4, 3, 4], for example).

This function attempts to kind of fix the problem of wrongly detected repeated commands.

For example, with the command (punctuation added for better understanding - remove it to test):
"turn on wifi and get the airplane mode on. no, don't turn the wifi on. turn off airplane mode and turn the wifi on.",
the command detection returns:
	"3234_wifi(),on \\// 3234_wifi(),on \\// 3234_wifi(),on \\// 3234_wifi(),on \\// 3234_wifi(),on \\//
	3234_airplane_mode(),on \\// 3234_airplane_mode(),on \\// 3234_wifi(),on \\// 3234_wifi(),on \\// 3234_wifi(),on
	\\// ".
Awfully wrong. This function improves that to "3234_wifi(),on \\// 3234_airplane_mode(),on \\// 3234_wifi(),on".
The first command is still wrong, the but idea here is to delete all the repeated elements (which improved MUCH in this
case).

Though, this also poses the problem of deleting purposefully repeated commands... Will be used until the
wordsVerificationFunction() can do the job better. In that case might be better (for now) to say the repeated commands in
another function call.

(As a curiosity, the overall Main() function can now know what to do in the example above, without needing to execute
at all!!! A thanks to this might be due to the new parameter on the wordsVerificationFunction() that ignores possibly
repeated commands!)
*/
func removeRepeatedCmds(ret_var string) string {
	var ret_var_list []string = strings.Split(ret_var, CMDS_SEPARATOR)

	const MARK_TERMINATION_STR string = "3234_MARK_TERMINATION_STR"

	var ret_var_list_len = len(ret_var_list) // Optimization
	for counter := 0; counter < ret_var_list_len; counter++ {
		if counter != ret_var_list_len-1 {
			if ret_var_list[counter+1] == ret_var_list[counter] {
				ret_var_list[counter] = MARK_TERMINATION_STR
			}
		}
	}

	ret_var = ""
	for _, command := range ret_var_list {
		if command != MARK_TERMINATION_STR {
			ret_var += command + CMDS_SEPARATOR
		}
	}

	if strings.HasSuffix(ret_var, CMDS_SEPARATOR) {
		ret_var = ret_var[:len(ret_var)-len(CMDS_SEPARATOR)]
	}

	return ret_var
}

// ATTENTION - none of these constants below can collide with the WARN_-started constants on CmdsListP1!!!
//const spec_cmd_dont_instead_CONST float32 = -1.1
//const spec_cmd_stop_CONST float32 = -2
//const spec_cmd_forget_CONST float32 = -3
const spec_cmd_dont_CONST float32 = -1

/*
sentenceCmdsDetector detects commands (whose indexes are listed in a slice of numbers) in a sentence of words.

-----------------------------------------------------------

> Params:

- sentence – a 1D slice of words on which the verification will be executed (basically it's sentence_str required by
Main() split by spaces in a 1D slice).

- allowed_cmds – same as in Main() but here it's in a slice of integers and not as a string


> Returns:

- a slice on which each index is a command found in the 'sentence', represented by one of its RET_-started constant
*/
func sentenceCmdsDetector(sentence []string, allowed_cmds []int) []float32 {
	var ret_var []float32 = nil

	for sentence_counter, sentence_word := range sentence {

		if "don't" == sentence_word {
			ret_var = append(ret_var, spec_cmd_dont_CONST)
		} else if WHATS_IT == sentence_word {
			float, _ := strconv.ParseFloat(WARN_WHATS_IT, 32)
			ret_var = append(ret_var, float32(float))
		} else {
			for _, cmd_index := range allowed_cmds {
				if cmd_index <= 0 {
					// Can't detect non-positive command identifiers. Those are reserved identifiers. So panic to warn
					// about wrong usage.
					GlobalUtilsInt_APU.PanicInt(1, "Non-positive command identifier sent for detection")
				} else if cmd_index > HIGHEST_CMD_INT {
					// Aside from getting the function to do additional tasks for nothing, there's nothing bad in
					// sending a command that is not on the list. But probably is bad practice to put commands from 1 to
					// 100 just to support all future commands - so panic.
					GlobalUtilsInt_APU.PanicInt(1, "Command identifier above highest value sent for detection")
				}

				for _, main_word := range main_words_GL[cmd_index] {
					if main_word == sentence_word {
						/*if cmd_index != 11 {
							// Uncomment for testing purposes
							continue
						}*/

						log.Println("==============")
						log.Println(sentence_word)
						log.Println(cmd_index)

						var results_WordsVerificationDADi [][]string = wordsVerificationFunction(sentence, sentence_counter,
							main_words_GL[cmd_index], words_list_GL[cmd_index], left_intervs_GL[cmd_index],
							right_intervs_GL[cmd_index], init_indexes_sub_verifs_GL[cmd_index],
							exclude_word_found_GL[cmd_index], return_last_match_GL[cmd_index],
							ignore_repets_main_words_GL[cmd_index], ignore_repets_cmds_GL[cmd_index],
							order_words_list_GL[cmd_index], stop_first_not_found_GL[cmd_index],
							exclude_original_words_GL[cmd_index], continue_with_words_slice_number_GL[cmd_index])

						log.Println("-----------")
						log.Println(results_WordsVerificationDADi)

						if checkResultsWordsVerifFunc(words_list_GL[cmd_index], sentence_word,
							results_WordsVerificationDADi, conditions_continue_GL[cmd_index],
							conditions_not_continue_GL[cmd_index]) {
							log.Println("LLLLLLL")
							var sub_cond_match_found bool = false
							for _, condition := range conditions_return_GL[cmd_index] {
								//log.Println("++++++++++++")
								if 1 == len(condition) {
									// Then it's check nothing of the results and just return immediately.
									float, _ := strconv.ParseFloat(condition[0][0], 32)
									ret_var = append(ret_var, float32(float))

									break
								} else {
									var all_sub_conds_matched bool = true
									var condition_len = len(condition) // Optimization
									for _, sub_cond := range condition[:condition_len-1] {
										// Here it's sub_condition_len-1 because the last one is the return constant.

										// Get the index of the results' sub-slice to check.
										sub_cond_index_to_chk, _ := strconv.Atoi(sub_cond[0])
										// If any word matches on the sub-condition, go check the next sub-condition.
										var word_match bool = false
										//log.Println("-------")
										for _, sub_cond_word := range sub_cond[1:] {
											// [1:] because sub_cond[0] is the index. The rest are word to check.

											//log.Println(word_1)
											if -1 == sub_cond_index_to_chk {
												// If the index of the results to check is -1, that means it's to check
												// the 'sentence_word' instead (the word that activated the command
												// detection).
												if sentence_word == sub_cond_word {
													word_match = true

													break
												}
											} else {
												//log.Println(results_WordsVerificationDADi[results_index][0])
												if results_WordsVerificationDADi[sub_cond_index_to_chk][0] == sub_cond_word {
													//log.Println("KKKKKKKKKKKKK")
													word_match = true

													break
												}
											}
										}
										all_sub_conds_matched = all_sub_conds_matched && word_match
										if !all_sub_conds_matched {
											// If any sub-condition had no match, forget about that condition and go
											// check the next one.
											break
										}
									}
									if all_sub_conds_matched {
										// In the end of a condition check, if all its sub-conditions found a match,
										// return the constant on the only index of the last sub-condition of the
										// condition.
										float, _ := strconv.ParseFloat(condition[condition_len-1][0], 32)
										ret_var = append(ret_var, float32(float))

										sub_cond_match_found = true
									}
								}
								if sub_cond_match_found {
									log.Println("QQQQQQQ")
									log.Println(ret_var)
									break
								}
							}
						}
					}
				}
			}
		}
	}

	return ret_var
}

/*
taskFilter filters a sentence of commands depending on special commands present on it.

For example, "turn on the lights and play some music. no, don't turn on the lights" --> the special command here is
"don't", this function will only leave on the slice the music command.

-----------------------------------------------------------

> Params:

- sentence_cmds – same as in sentenceCmdsDetector()


> Returns:

- nothing
*/
func taskFilter(sentence_cmds *[]float32) {
	// For testing
	//*sentence_filtered = [][]string{{"test"}, {"test"}, {"test 234 lkj"}, {"test"}, {"test"}, {"test"}, {"test"},
	//	{"test"}, {"test"}, {"test"}, {"test"}, {"test"}, {"test"}, {"test"}, {"test"}, }
	//*sentence_cmds = []float32{24, -1, 26, 25, -1, -1, -1, 25, 24}

	log.Println("==============================================")
	log.Println("*sentence_cmds -->", *sentence_cmds)

	// RESTRICTED VALUE ON THE sentence_cmds SLICE - Used to mark elements for deletion on the slice. This way, they're
	// deleted only in the end and on the main loop it doesn't get confusing about which elements have been deleted
	// already.
	const MARK_TERMINATION_FLOAT32 float32 = 0

	for counter, number := range *sentence_cmds {
		if spec_cmd_dont_CONST == number {

			var delete_number_before_dont bool = false

			// Delete the "don't"
			(*sentence_cmds)[counter] = MARK_TERMINATION_FLOAT32

			//log.Println("1 -", *sentence_cmds)
			if counter != len(*sentence_cmds)-1 {
				// If the next index is within the maximum index (which means, if the next number exists)...

				var next_number float32 = (*sentence_cmds)[counter+1]
				if next_number > 0 { // Means if it's a normal command. If it is, assume the below case.
					// Case: "do [1] and do [2]. no don't do [1]" - delete this, don't, and this. Also, if by any reason
					// there are more copies of [1], delete them also - if they're before the next element only.

					var number_mentioned bool = false
					var pos_next_number []int = nil
					for counter1, number1 := range *sentence_cmds {
						if number1 == next_number {
							pos_next_number = append(pos_next_number, counter1)
							number_mentioned = true
						}
						if counter1 == counter {
							// Stop when it gets to before the next element
							break
						}
					}
					if number_mentioned {
						// If the number was mentioned before (like [24, 25, 24, -1, 24]), delete all copies and the -1.
						(*sentence_cmds)[counter+1] = MARK_TERMINATION_FLOAT32

						//log.Println("2 -", *sentence_cmds)

						for _, index_element := range pos_next_number {
							(*sentence_cmds)[index_element] = MARK_TERMINATION_FLOAT32
						}
						//log.Println("3 -", *sentence_cmds)
					} else {
						// Else, delete only the element before the current "don't" (if there exists one).
						// Example: [24, -1, 26, 25, -1, 25, 24] will become [26, 24], because, "do 24, no don't do
						// it. do 26 and do 25. no don't do 25. do 24."
						delete_number_before_dont = true
					}
				}
				// Else, if it's not a positive number, assume the below case.
				// Case: "do this, no don't do it, don't do it. do that". Delete only the current "don't" as was done
				// above and keep doing it (the loop will automatically) until there's only one, which will be the one
				// used to decide what to delete (done above).
			} else {
				// If there's no more elements, there can be previous ones. So delete the previous number to the "don't".
				// Which would be a "do [1]. no, never mind, don't do it".
				delete_number_before_dont = true
			}

			if delete_number_before_dont {
				// Do it only if there's a normal command before. If it's for example WARN_WHATS_IT, don't delete it.
				if counter-1 >= 0 && (*sentence_cmds)[counter-1] > 0 {
					(*sentence_cmds)[counter-1] = MARK_TERMINATION_FLOAT32
					//log.Println("4 -", *sentence_cmds)
				}
			}
		}
	}

	//log.Println("5 -", *sentence_cmds)

	// Delete all elements marked for deletion
	for counter := 0; counter < len(*sentence_cmds); counter++ {
		// Don't forget (again) the length is checked every time on the loop
		if MARK_TERMINATION_FLOAT32 == (*sentence_cmds)[counter] {
			GlobalUtilsInt_APU.DelElemInSlice(sentence_cmds, counter)
			counter--
		}
	}

	log.Println("*sentence_cmds -->", *sentence_cmds)
	log.Println("==============================================")
}
